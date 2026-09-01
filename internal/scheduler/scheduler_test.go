package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
)

// newTestScheduler starts an embedded bus and returns the scheduler and the
// publisher (for subscribing to the subjects it publishes on).
func newTestScheduler(t *testing.T) (*Scheduler, publisher.Publisher) {
	t.Helper()
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := store.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	pub, err := publisher.ConnectOn(0)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return New(db, pub, 5*time.Second), pub
}

// subscribe opens a JetStream subscription on subject, forwarding messages to
// a channel. Each call uses a unique durable name.
func subscribe(t *testing.T, pub publisher.Publisher, subject, name string) (<-chan publisher.Message, func()) {
	t.Helper()
	ch := make(chan publisher.Message, 32)
	sub, err := pub.Subscribe(context.Background(), subject, name, func(m publisher.Message) {
		ch <- m
		_ = m.Ack()
	})
	if err != nil {
		t.Fatalf("Subscribe(%s): %v", subject, err)
	}
	return ch, sub.Stop
}

func TestFireWithinGrace(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx := context.Background()

	got, stop := subscribe(t, pub, "scheduler.test", "fire")
	defer stop()
	id, err := s.Schedule(ctx, "scheduler.test", []byte(`{"hi":1}`), time.Now())
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if id == "" {
		t.Fatal("empty id")
	}

	if err := s.poll(ctx); err != nil {
		t.Fatalf("poll: %v", err)
	}

	select {
	case m := <-got:
		if string(m.Data()) != `{"hi":1}` {
			t.Errorf("payload = %s, want {\"hi\":1}", m.Data())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("message never published")
	}
}

func TestExpirePastGrace(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx := context.Background()

	expired, stopExp := subscribe(t, pub, events.SchedulerExpired{}.Subject(), "expired")
	defer stopExp()
	payload, stopPayload := subscribe(t, pub, "scheduler.test", "payload")
	defer stopPayload()

	id, err := s.Schedule(ctx, "scheduler.test", []byte(`{"hi":1}`), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	// Make the deadline long past, beyond the grace period.
	if _, err := s.db.ExecContext(ctx, `UPDATE scheduler SET publish_at = ?
		WHERE id = ?`, time.Now().Add(-time.Minute).UnixMilli(), id); err != nil {
		t.Fatalf("rewind deadline: %v", err)
	}

	if err := s.poll(ctx); err != nil {
		t.Fatalf("poll: %v", err)
	}

	select {
	case m := <-expired:
		var notice events.SchedulerExpired
		if err := json.Unmarshal(m.Data(), &notice); err != nil {
			t.Fatalf("unmarshal notice: %v", err)
		}
		if notice.ID != id {
			t.Errorf("notice id = %q, want %q", notice.ID, id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expired notice never published")
	}

	// The payload must not have been published on its subject.
	select {
	case m := <-payload:
		t.Errorf("expired message was published on its subject: %s", m.Data())
	case <-time.After(50 * time.Millisecond):
	}
}

func TestReplayUnpublishedAfterClaim(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx := context.Background()

	row, err := store.CreateScheduled(ctx, s.db, "scheduler.test", `{"x":2}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	// Simulate a crash between claiming and publishing.
	if won, err := store.ClaimForFire(ctx, s.db, row.ID); err != nil || !won {
		t.Fatalf("ClaimForFire: won=%v err=%v", won, err)
	}

	got, stop := subscribe(t, pub, "scheduler.test", "replay")
	defer stop()
	if err := s.poll(ctx); err != nil {
		t.Fatalf("poll: %v", err)
	}

	select {
	case m := <-got:
		if string(m.Data()) != `{"x":2}` {
			t.Errorf("payload = %s, want {\"x\":2}", m.Data())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("unpublished message never replayed")
	}
}

func TestScheduleFutureFiresOnTimer(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx := context.Background()

	got, stop := subscribe(t, pub, "scheduler.test", "timer")
	defer stop()
	id, err := s.Schedule(ctx, "scheduler.test", []byte(`{"t":1}`), time.Now().Add(300*time.Millisecond))
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if id == "" {
		t.Fatal("empty id")
	}

	// No poll call: the armed timer alone must deliver at the future deadline.
	select {
	case m := <-got:
		if string(m.Data()) != `{"t":1}` {
			t.Errorf("payload = %s, want {\"t\":1}", m.Data())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timer never fired")
	}
}

func TestCancelWithdrawsPending(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx := context.Background()

	got, stop := subscribe(t, pub, "scheduler.test", "cancel")
	defer stop()
	id, err := s.Schedule(ctx, "scheduler.test", []byte(`{"x":1}`), time.Now().Add(300*time.Millisecond))
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	won, err := s.Cancel(ctx, id)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if !won {
		t.Fatal("cancel of a pending message must win")
	}
	// A second cancel loses: nothing left to withdraw.
	won, err = s.Cancel(ctx, id)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if won {
		t.Fatal("second cancel must lose")
	}

	// The armed timer was stopped; a poll must not fire it either, even after
	// the original deadline passes.
	time.Sleep(400 * time.Millisecond)
	if err := s.poll(ctx); err != nil {
		t.Fatalf("poll: %v", err)
	}
	select {
	case m := <-got:
		t.Errorf("cancelled message was published: %s", m.Data())
	case <-time.After(100 * time.Millisecond):
	}

	var status string
	if err := s.db.QueryRowContext(ctx,
		`SELECT status FROM scheduler WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if status != "cancelled" {
		t.Errorf("status = %q, want cancelled", status)
	}
}

func TestCancelFiredLoses(t *testing.T) {
	s, _ := newTestScheduler(t)
	ctx := context.Background()

	id, err := s.Schedule(ctx, "scheduler.test", []byte(`{"x":1}`), time.Now())
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if err := s.poll(ctx); err != nil {
		t.Fatalf("poll: %v", err)
	}
	won, err := s.Cancel(ctx, id)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if won {
		t.Fatal("cancel of a fired message must lose")
	}
}

func TestListScheduledViaScheduler(t *testing.T) {
	s, _ := newTestScheduler(t)
	ctx := context.Background()

	first, err := s.Schedule(ctx, "agent.run", []byte(`{"p":1}`), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	second, err := s.Schedule(ctx, "agent.run", []byte(`{"p":2}`), time.Now().Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if won, err := s.Cancel(ctx, second); err != nil || !won {
		t.Fatalf("Cancel: won=%v err=%v", won, err)
	}

	rows, err := s.ListScheduled(ctx)
	if err != nil {
		t.Fatalf("ListScheduled: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("ListScheduled = %d rows, want 2", len(rows))
	}
	// Pending first (soonest deadline first), then terminal rows by deadline.
	if rows[0].ID != first || rows[0].Status != "pending" {
		t.Errorf("rows[0] = %+v, want the pending %s", rows[0], first)
	}
	if rows[1].ID != second || rows[1].Status != "cancelled" {
		t.Errorf("rows[1] = %+v, want the cancelled %s", rows[1], second)
	}
}

func TestScheduleRecurringFiresRepeatedly(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx := context.Background()
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	const interval = 150 * time.Millisecond
	got, stop := subscribe(t, pub, "scheduler.test", "rec")
	defer stop()
	go s.Run(runCtx)

	id, err := s.ScheduleRecurring(ctx, "scheduler.test", []byte(`{"r":1}`), interval)
	if err != nil {
		t.Fatalf("ScheduleRecurring: %v", err)
	}
	if id == "" {
		t.Fatal("empty recurrence id")
	}

	// Three occurrences must arrive on their own timers, without any poll.
	const want = 3
	collect := time.After(10 * time.Second)
	seen := 0
	for seen < want {
		select {
		case m := <-got:
			if string(m.Data()) != `{"r":1}` {
				t.Errorf("payload = %s, want {\"r\":1}", m.Data())
			}
			seen++
		case <-collect:
			t.Fatalf("received %d/%d occurrences", seen, want)
		}
	}

	// The schedule is still active with a pending next occurrence.
	recs, err := s.ListRecurrences(ctx)
	if err != nil {
		t.Fatalf("ListRecurrences: %v", err)
	}
	if len(recs) != 1 || recs[0].Status != "active" {
		t.Errorf("recurrences = %+v, want one active", recs)
	}
	var pending int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM scheduler WHERE recurrence_id = ? AND status = 'pending'`, id).
		Scan(&pending); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pending != 1 {
		t.Errorf("pending occurrences = %d, want 1", pending)
	}
}

func TestScheduleRecurringCancelStopsFutureOccurrences(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx := context.Background()

	const interval = 100 * time.Millisecond
	got, stop := subscribe(t, pub, "scheduler.test", "rec-cancel")
	defer stop()

	id, err := s.ScheduleRecurring(ctx, "scheduler.test", []byte(`{"r":1}`), interval)
	if err != nil {
		t.Fatalf("ScheduleRecurring: %v", err)
	}

	// The first occurrence fires on its own timer.
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("first occurrence never fired")
	}

	won, err := s.Cancel(ctx, id)
	if err != nil {
		t.Fatalf("Cancel recurrence: %v", err)
	}
	if !won {
		t.Fatal("cancel of an active recurrence must win")
	}
	// A second cancel loses.
	won, err = s.Cancel(ctx, id)
	if err != nil {
		t.Fatalf("Cancel recurrence: %v", err)
	}
	if won {
		t.Fatal("second cancel must lose")
	}

	// Wait past several deadlines; the armed next-occurrence timer fires but
	// the claim loses because the child was cancelled, so nothing publishes.
	time.Sleep(300 * time.Millisecond)
	if err := s.poll(ctx); err != nil {
		t.Fatalf("poll: %v", err)
	}
	select {
	case m := <-got:
		t.Errorf("occurrence after cancel: %s", m.Data())
	case <-time.After(200 * time.Millisecond):
	}

	var status string
	if err := s.db.QueryRowContext(ctx,
		`SELECT status FROM recurrences WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatalf("read recurrence: %v", err)
	}
	if status != "cancelled" {
		t.Errorf("recurrence status = %q, want cancelled", status)
	}
	var pending int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM scheduler WHERE recurrence_id = ? AND status = 'pending'`, id).
		Scan(&pending); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pending != 0 {
		t.Errorf("pending occurrences = %d, want 0", pending)
	}
}

func TestRecurringMissedPublishesEvent(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx := context.Background()

	missed, stopMissed := subscribe(t, pub, events.RecurringMissed{}.Subject(), "rec-missed")
	defer stopMissed()
	payload, stopPayload := subscribe(t, pub, "scheduler.test", "rec-missed-payload")
	defer stopPayload()

	id, err := s.ScheduleRecurring(ctx, "scheduler.test", []byte(`{"r":1}`), time.Minute)
	if err != nil {
		t.Fatalf("ScheduleRecurring: %v", err)
	}
	// Rewind the first occurrence's deadline far past the grace period.
	past := time.Now().Add(-time.Minute)
	var firstID string
	if err := s.db.QueryRowContext(ctx,
		`SELECT id FROM scheduler WHERE recurrence_id = ? AND status = 'pending'`, id).Scan(&firstID); err != nil {
		t.Fatalf("find first occurrence: %v", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE scheduler SET publish_at = ? WHERE id = ?`, past.UnixMilli(), firstID); err != nil {
		t.Fatalf("rewind deadline: %v", err)
	}

	if err := s.poll(ctx); err != nil {
		t.Fatalf("poll: %v", err)
	}

	select {
	case m := <-missed:
		var e events.RecurringMissed
		if err := json.Unmarshal(m.Data(), &e); err != nil {
			t.Fatalf("unmarshal notice: %v", err)
		}
		if e.RecurrenceID != id {
			t.Errorf("notice recurrence_id = %q, want %q", e.RecurrenceID, id)
		}
		if e.OccurredAt != past.UnixMilli() {
			t.Errorf("notice occurred_at = %d, want %d", e.OccurredAt, past.UnixMilli())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("missed notice never published")
	}
	// The skipped occurrence's payload must not be published on its subject.
	select {
	case m := <-payload:
		t.Errorf("skipped occurrence was published: %s", m.Data())
	case <-time.After(100 * time.Millisecond):
	}
	// The schedule advanced: a pending next occurrence exists and the
	// recurrence is still active.
	var pending int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM scheduler WHERE recurrence_id = ? AND status = 'pending'`, id).
		Scan(&pending); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pending != 1 {
		t.Errorf("pending occurrences = %d, want 1 (advanced)", pending)
	}
	var status string
	if err := s.db.QueryRowContext(ctx,
		`SELECT status FROM recurrences WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatalf("read recurrence: %v", err)
	}
	if status != "active" {
		t.Errorf("recurrence status = %q, want active", status)
	}
}

func TestScheduleRecurringRejectsZeroInterval(t *testing.T) {
	s, _ := newTestScheduler(t)
	if _, err := s.ScheduleRecurring(context.Background(), "x", []byte(`{}`), 0); err == nil {
		t.Fatal("ScheduleRecurring with zero interval did not error")
	}
}

func TestRunExpiresPastDeadlineOnStartup(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx := context.Background()

	row, err := store.CreateScheduled(ctx, s.db, "scheduler.test", `{"old":1}`,
		time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	expired, stopExp := subscribe(t, pub, events.SchedulerExpired{}.Subject(), "run-expired")
	defer stopExp()
	payload, stopPayload := subscribe(t, pub, "scheduler.test", "run-payload")
	defer stopPayload()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go s.Run(runCtx)

	select {
	case m := <-expired:
		var notice events.SchedulerExpired
		if err := json.Unmarshal(m.Data(), &notice); err != nil {
			t.Fatalf("unmarshal notice: %v", err)
		}
		if notice.ID != row.ID {
			t.Errorf("notice id = %q, want %q", notice.ID, row.ID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("expired notice never published")
	}
	select {
	case m := <-payload:
		t.Errorf("expired message was published on its subject: %s", m.Data())
	case <-time.After(100 * time.Millisecond):
	}

	var status string
	if err := s.db.QueryRowContext(ctx,
		`SELECT status FROM scheduler WHERE id = ?`, row.ID).Scan(&status); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if status != "expired" {
		t.Errorf("status = %q, want expired", status)
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	s, _ := newTestScheduler(t)
	ctx, cancel := context.WithCancel(context.Background())

	if _, err := s.Schedule(context.Background(), "x", []byte(`{}`),
		time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	// Give Run time to finish its startup scan and enter the loop.
	time.Sleep(100 * time.Millisecond)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error on cancel: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}

func TestManyScheduleExactlyOnce(t *testing.T) {
	s, pub := newTestScheduler(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const n = 50
	got, stop := subscribe(t, pub, "scheduler.test", "many")
	defer stop()

	go s.Run(ctx)

	publishAt := time.Now().Add(50 * time.Millisecond)
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		id, err := s.Schedule(ctx, "scheduler.test", []byte(fmt.Sprintf(`{"i":%d}`, i)), publishAt)
		if err != nil {
			t.Fatalf("Schedule %d: %v", i, err)
		}
		ids[i] = id
	}

	// Every scheduled message must be delivered exactly once.
	received := make(map[string]bool)
	collectDeadline := time.After(10 * time.Second)
	for len(received) < n {
		select {
		case m := <-got:
			data := string(m.Data())
			if received[data] {
				t.Errorf("duplicate payload %q", data)
			}
			received[data] = true
		case <-collectDeadline:
			t.Fatalf("received %d/%d messages", len(received), n)
		}
	}

	// Wait until the replay scan can no longer republish anything, then assert
	// no duplicate arrives.
	markDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(markDeadline) {
		var unmarked int
		if err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM scheduler WHERE status = 'fired' AND published_at IS NULL`).
			Scan(&unmarked); err != nil {
			t.Fatalf("count unmarked: %v", err)
		}
		if unmarked == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case m := <-got:
		t.Errorf("duplicate publish after settlement: %q", m.Data())
	case <-time.After(300 * time.Millisecond):
	}

	for _, id := range ids {
		var status string
		var publishedAt sql.NullString
		if err := s.db.QueryRowContext(ctx,
			`SELECT status, published_at FROM scheduler WHERE id = ?`, id).
			Scan(&status, &publishedAt); err != nil {
			t.Fatalf("read row %s: %v", id, err)
		}
		if status != "fired" {
			t.Errorf("row %s status = %q, want fired", id, status)
		}
		if !publishedAt.Valid {
			t.Errorf("row %s has no published_at", id)
		}
	}
}
