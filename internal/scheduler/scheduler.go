package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
)

const (
	// pollInterval is how often the backstop scan runs. Timers provide exact
	// deadlines; the poll catches what the timers missed after a restart or a
	// crash.
	pollInterval = time.Second

	// defaultGrace is how far past its deadline a message may be found before
	// it is expired instead of fired. A few seconds absorbs the timer-vs-poll
	// race at a deadline without turning real downtime into normal deliveries.
	defaultGrace = 5 * time.Second

	// publishTimeout bounds each bus publish so a stalled bus cannot wedge a
	// timer or the poll loop.
	publishTimeout = 10 * time.Second
)

// Scheduler durably fires one-time messages: rows live in SQLite, per-message
// timers give low-latency delivery, and a poll backstop replays or expires
// anything the timers missed. Scheduler is safe for concurrent use.
//
// Delivery is exactly-once: every publish carries the message's row id as its
// dedup id, so if the process crashes between publishing and recording the
// publish, the replay on restart is dropped by the bus's dedup window instead
// of being delivered again.
type Scheduler struct {
	db  *sql.DB
	pub publisher.Publisher
	// grace is the deadline slack before a due message is expired instead of
	// fired.
	grace time.Duration

	mu     sync.Mutex
	timers map[string]*time.Timer
}

// New returns a Scheduler backed by db, publishing through pub. A non-positive
// grace falls back to defaultGrace.
func New(db *sql.DB, pub publisher.Publisher, grace time.Duration) *Scheduler {
	if grace <= 0 {
		grace = defaultGrace
	}
	return &Scheduler{db: db, pub: pub, grace: grace, timers: make(map[string]*time.Timer)}
}

// Schedule durably records a one-time message and arms a timer for it. The id
// and the row's creation time are both derived from a fresh UUIDv7; the payload
// is published once publishAt passes. It returns the message id.
func (s *Scheduler) Schedule(ctx context.Context, subject string, payload []byte, publishAt time.Time) (string, error) {
	row, err := store.CreateScheduled(ctx, s.db, subject, string(payload), publishAt)
	if err != nil {
		return "", err
	}
	s.arm(*row)
	return row.ID, nil
}

// ScheduleRecurring durably records a fixed-interval repeating schedule and
// arms its first occurrence, which fires at now + interval. Each occurrence is
// its own scheduler row, so delivery keeps the one-shot exactly-once machinery;
// the schedule advances whenever an occurrence fires or is skipped. It returns
// the recurrence id. Cancel it with Cancel.
func (s *Scheduler) ScheduleRecurring(ctx context.Context, subject string, payload []byte, interval time.Duration) (string, error) {
	if interval <= 0 {
		return "", fmt.Errorf("schedule recurring: interval must be positive")
	}
	rec, err := store.CreateRecurrence(ctx, s.db, subject, string(payload), interval)
	if err != nil {
		return "", err
	}
	row, err := store.CreateScheduledOccurrence(ctx, s.db, subject, string(payload), time.Now().Add(interval), rec.ID)
	if err != nil {
		// Best-effort cleanup: without a first occurrence the schedule would
		// never advance, so withdraw it rather than leave an orphan.
		_, _ = store.CancelRecurrence(ctx, s.db, rec.ID)
		return "", err
	}
	s.arm(*row)
	return rec.ID, nil
}

// Cancel withdraws a scheduled message or a whole recurrence so it will never
// fire. For a scheduler row it stops the timer and flips the row from pending
// to cancelled. For a recurrence id it cancels the schedule and its pending
// next occurrence. It reports whether anything was cancelled; a false result
// means the id was not a pending run or an active recurrence.
func (s *Scheduler) Cancel(ctx context.Context, id string) (bool, error) {
	s.mu.Lock()
	if t, ok := s.timers[id]; ok {
		t.Stop()
		delete(s.timers, id)
	}
	s.mu.Unlock()
	if won, err := store.CancelScheduled(ctx, s.db, id); err != nil {
		return false, err
	} else if won {
		return true, nil
	}
	return store.CancelRecurrence(ctx, s.db, id)
}

// ListScheduled returns every scheduler row, pending first (soonest deadline
// first), then the terminal rows by deadline. It gives callers the full view
// needed to decide which scheduled messages to cancel.
func (s *Scheduler) ListScheduled(ctx context.Context) ([]store.Scheduled, error) {
	return store.ListScheduled(ctx, s.db)
}

// ListRecurrences returns every recurrence, active first, then cancelled.
func (s *Scheduler) ListRecurrences(ctx context.Context) ([]store.Recurrence, error) {
	return store.ListRecurrences(ctx, s.db)
}

// Run serves the scheduler until ctx is cancelled: it arms timers for pending
// messages on startup, then polls periodically to fire or expire what the
// timers missed. It returns nil on cancellation and the first unrecoverable
// error otherwise.
func (s *Scheduler) Run(ctx context.Context) error {
	pending, err := store.ListPending(ctx, s.db)
	if err != nil {
		return err
	}
	for _, row := range pending {
		if time.Until(time.UnixMilli(row.PublishAt)) > 0 {
			s.arm(row)
		}
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			for _, t := range s.timers {
				t.Stop()
			}
			s.mu.Unlock()
			return nil
		case <-ticker.C:
			if err := s.poll(ctx); err != nil {
				return err
			}
		}
	}
}

func (s *Scheduler) arm(row store.Scheduled) {
	d := time.Until(time.UnixMilli(row.PublishAt))
	if d <= 0 {
		return
	}
	timer := time.AfterFunc(d, func() {
		s.mu.Lock()
		delete(s.timers, row.ID)
		s.mu.Unlock()
		s.fire(context.Background(), row)
	})
	s.mu.Lock()
	s.timers[row.ID] = timer
	s.mu.Unlock()
}

// Poll runs one backstop scan: it fires or expires due messages and replays
// messages that were fired but never marked published. Run calls it
// periodically; Poll is exposed so external callers can trigger a scan on
// demand.
func (s *Scheduler) Poll(ctx context.Context) error {
	return s.poll(ctx)
}

// poll is the backstop scan: it fires or expires due messages, then replays
// fired messages that were never marked published.
func (s *Scheduler) poll(ctx context.Context) error {
	now := time.Now().UnixMilli()
	due, err := store.ListDue(ctx, s.db, now)
	if err != nil {
		return err
	}
	for _, row := range due {
		if now-row.PublishAt <= s.grace.Milliseconds() {
			if err := s.fire(ctx, row); err != nil {
				return err
			}
		} else if err := s.expire(ctx, row); err != nil {
			return err
		}
	}

	unpublished, err := store.ListUnpublished(ctx, s.db)
	if err != nil {
		return err
	}
	for _, row := range unpublished {
		if err := s.publishAndMark(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// fire claims a pending message and, if this process won the claim, publishes
// its payload and records the publish. A won claim on a recurring occurrence
// also schedules its next occurrence (the claim and the advance are one
// transaction), so exactly one process moves the schedule forward.
func (s *Scheduler) fire(ctx context.Context, row store.Scheduled) error {
	won, next, err := store.ClaimAndAdvance(ctx, s.db, row.ID, "fired")
	if err != nil {
		return err
	}
	if !won {
		return nil
	}
	if next != nil {
		s.arm(*next)
	}
	return s.publishAndMark(ctx, row)
}

// expire claims a pending message found far past its deadline and, if this
// process won the claim, announces it. A recurring occurrence is skipped: its
// next occurrence is scheduled (claim and advance are one transaction) and the
// miss is announced on scheduler.recurring.missed. A one-shot message is
// announced on scheduler.expired. The notices are best-effort; the message
// stays expired either way.
func (s *Scheduler) expire(ctx context.Context, row store.Scheduled) error {
	won, next, err := store.ClaimAndAdvance(ctx, s.db, row.ID, "expired")
	if err != nil {
		return err
	}
	if !won {
		return nil
	}
	if next != nil {
		s.arm(*next)
	}
	if row.RecurrenceID.Valid {
		return s.pub.Publish(ctx, events.RecurringMissed{
			RecurrenceID: row.RecurrenceID.String,
			OccurredAt:   row.PublishAt,
		})
	}
	return s.pub.Publish(ctx, events.SchedulerExpired{ID: row.ID})
}

// publishAndMark sends the payload on the message's subject with the message's
// id as its dedup id, then records the publish so the replay scan leaves it
// alone. A crash between the publish and the mark replays the same dedup id,
// which the bus's dedup window drops, keeping delivery exactly-once.
func (s *Scheduler) publishAndMark(ctx context.Context, row store.Scheduled) error {
	pubCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	if err := s.pub.PublishMsg(pubCtx, row.Subject, []byte(row.Payload), row.ID); err != nil {
		return err
	}
	return store.MarkPublished(ctx, s.db, row.ID)
}
