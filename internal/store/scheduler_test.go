package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newSchedulerTestDB(t *testing.T) (*sql.DB, context.Context) {
	t.Helper()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db, context.Background()
}

func TestCreateScheduledRecordsTimes(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	publishAt := time.Now().Add(30 * time.Minute)
	s, err := CreateScheduled(ctx, db, "foo.bar", `{"hi":1}`, publishAt)
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	if s.ID == "" {
		t.Fatal("empty id")
	}
	if s.Status != "pending" {
		t.Errorf("status = %q, want pending", s.Status)
	}
	if s.PublishedAt.Valid {
		t.Error("published_at set before publish")
	}
	// created_at mirrors the timestamp embedded in the id.
	u, err := uuid.Parse(s.ID)
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}
	if want := unixMillisFromUUIDV7(u).UTC().String(); s.CreatedAt != want {
		t.Errorf("created_at = %q, want uuid-derived %q", s.CreatedAt, want)
	}
	// publish_at is the caller-supplied target, not the insertion time.
	if s.PublishAt != publishAt.UnixMilli() {
		t.Errorf("publish_at = %d, want %d", s.PublishAt, publishAt.UnixMilli())
	}
}

func TestClaimForFireAndMarkPublished(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	s, err := CreateScheduled(ctx, db, "foo.bar", `{}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}

	won, err := ClaimForFire(ctx, db, s.ID)
	if err != nil {
		t.Fatalf("ClaimForFire: %v", err)
	}
	if !won {
		t.Fatal("first claim should win")
	}
	// A second claim loses: the timer-vs-poll race must publish once.
	won, err = ClaimForFire(ctx, db, s.ID)
	if err != nil {
		t.Fatalf("ClaimForFire: %v", err)
	}
	if won {
		t.Fatal("second claim must lose")
	}

	if err := MarkPublished(ctx, db, s.ID); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}
	got, err := ListUnpublished(ctx, db)
	if err != nil {
		t.Fatalf("ListUnpublished: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("fired+published row still listed as unpublished: %+v", got)
	}
}

func TestClaimExpiredOnlyFromPending(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	s, err := CreateScheduled(ctx, db, "foo.bar", `{}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}

	won, err := ClaimExpired(ctx, db, s.ID)
	if err != nil {
		t.Fatalf("ClaimExpired: %v", err)
	}
	if !won {
		t.Fatal("first claim should win")
	}
	won, err = ClaimExpired(ctx, db, s.ID)
	if err != nil {
		t.Fatalf("ClaimExpired: %v", err)
	}
	if won {
		t.Fatal("second claim must lose")
	}
	// Fired rows cannot be claimed as expired either.
	if _, err := ClaimForFire(ctx, db, s.ID); err != nil {
		t.Fatalf("ClaimForFire on expired: %v", err)
	}
}

func TestListDue(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	s, err := CreateScheduled(ctx, db, "foo.bar", `{}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}

	if got, err := ListDue(ctx, db, s.PublishAt-1); err != nil {
		t.Fatalf("ListDue: %v", err)
	} else if len(got) != 0 {
		t.Errorf("ListDue before deadline = %d rows, want 0", len(got))
	}
	got, err := ListDue(ctx, db, s.PublishAt)
	if err != nil {
		t.Fatalf("ListDue: %v", err)
	}
	if len(got) != 1 || got[0].ID != s.ID {
		t.Errorf("ListDue at deadline = %+v, want the scheduled row", got)
	}
}

func TestListPending(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	late, err := CreateScheduled(ctx, db, "a", `{}`, time.Now().Add(2*time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	soon, err := CreateScheduled(ctx, db, "b", `{}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	expired, err := CreateScheduled(ctx, db, "c", `{}`, time.Now().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	fired, err := CreateScheduled(ctx, db, "d", `{}`, time.Now().Add(4*time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	if won, err := ClaimExpired(ctx, db, expired.ID); err != nil || !won {
		t.Fatalf("ClaimExpired: won=%v err=%v", won, err)
	}
	if won, err := ClaimForFire(ctx, db, fired.ID); err != nil || !won {
		t.Fatalf("ClaimForFire: won=%v err=%v", won, err)
	}

	got, err := ListPending(ctx, db)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListPending = %d rows, want 2 (fired and expired excluded)", len(got))
	}
	if got[0].ID != soon.ID || got[1].ID != late.ID {
		t.Errorf("ListPending order = [%s, %s], want [%s, %s] (soonest deadline first)",
			got[0].ID, got[1].ID, soon.ID, late.ID)
	}
}

func TestMarkPublishedRecordsTime(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	s, err := CreateScheduled(ctx, db, "foo.bar", `{}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	if _, err := ClaimForFire(ctx, db, s.ID); err != nil {
		t.Fatalf("ClaimForFire: %v", err)
	}
	if err := MarkPublished(ctx, db, s.ID); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}

	var status string
	var publishedAt sql.NullString
	if err := db.QueryRowContext(ctx,
		`SELECT status, published_at FROM scheduler WHERE id = ?`, s.ID).
		Scan(&status, &publishedAt); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if status != "fired" {
		t.Errorf("status = %q, want fired", status)
	}
	if !publishedAt.Valid || publishedAt.String == "" {
		t.Errorf("published_at = %+v, want a timestamp", publishedAt)
	}
}

func TestCancelScheduledOnlyFromPending(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	s, err := CreateScheduled(ctx, db, "foo.bar", `{}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}

	won, err := CancelScheduled(ctx, db, s.ID)
	if err != nil {
		t.Fatalf("CancelScheduled: %v", err)
	}
	if !won {
		t.Fatal("first cancel should win")
	}
	// A second cancel loses: the row is already cancelled.
	won, err = CancelScheduled(ctx, db, s.ID)
	if err != nil {
		t.Fatalf("CancelScheduled: %v", err)
	}
	if won {
		t.Fatal("second cancel must lose")
	}

	var status string
	if err := db.QueryRowContext(ctx,
		`SELECT status FROM scheduler WHERE id = ?`, s.ID).Scan(&status); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if status != "cancelled" {
		t.Errorf("status = %q, want cancelled", status)
	}
}

func TestCancelScheduledSkipsTerminalRows(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	for _, tc := range []struct {
		name   string
		claim  func(id string) (bool, error)
		status string
	}{
		{"fired", func(id string) (bool, error) { return ClaimForFire(ctx, db, id) }, "fired"},
		{"expired", func(id string) (bool, error) { return ClaimExpired(ctx, db, id) }, "expired"},
		{"cancelled", func(id string) (bool, error) { return CancelScheduled(ctx, db, id) }, "cancelled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := CreateScheduled(ctx, db, "foo.bar", `{}`, time.Now().Add(time.Minute))
			if err != nil {
				t.Fatalf("CreateScheduled: %v", err)
			}
			if won, err := tc.claim(s.ID); err != nil || !won {
				t.Fatalf("seed claim: won=%v err=%v", won, err)
			}
			won, err := CancelScheduled(ctx, db, s.ID)
			if err != nil {
				t.Fatalf("CancelScheduled: %v", err)
			}
			if won {
				t.Errorf("cancel won on a %s row", tc.status)
			}
			var status string
			if err := db.QueryRowContext(ctx,
				`SELECT status FROM scheduler WHERE id = ?`, s.ID).Scan(&status); err != nil {
				t.Fatalf("read row: %v", err)
			}
			if status != tc.status {
				t.Errorf("status = %q, want %q", status, tc.status)
			}
		})
	}
}

func TestListScheduledOrdersPendingFirst(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	early, err := CreateScheduled(ctx, db, "a", `{"p":1}`, time.Now().Add(2*time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	soon, err := CreateScheduled(ctx, db, "b", `{"p":2}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	fired, err := CreateScheduled(ctx, db, "c", `{"p":3}`, time.Now().Add(4*time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	cancelled, err := CreateScheduled(ctx, db, "d", `{"p":4}`, time.Now().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	if won, err := ClaimForFire(ctx, db, fired.ID); err != nil || !won {
		t.Fatalf("ClaimForFire: won=%v err=%v", won, err)
	}
	if won, err := CancelScheduled(ctx, db, cancelled.ID); err != nil || !won {
		t.Fatalf("CancelScheduled: won=%v err=%v", won, err)
	}

	got, err := ListScheduled(ctx, db)
	if err != nil {
		t.Fatalf("ListScheduled: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("ListScheduled = %d rows, want 4", len(got))
	}
	// Pending first (soonest deadline first), then terminal rows by deadline.
	want := []string{soon.ID, early.ID, cancelled.ID, fired.ID}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("ListScheduled[%d] = %s, want %s", i, got[i].ID, id)
		}
	}
}

func TestCreateScheduledOccurrenceLinksRecurrence(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	rec, err := CreateRecurrence(ctx, db, "foo.bar", `{"p":1}`, 5*time.Minute)
	if err != nil {
		t.Fatalf("CreateRecurrence: %v", err)
	}
	at := time.Now().Add(5 * time.Minute)
	s, err := CreateScheduledOccurrence(ctx, db, rec.Subject, rec.Payload, at, rec.ID)
	if err != nil {
		t.Fatalf("CreateScheduledOccurrence: %v", err)
	}
	if s.Status != "pending" {
		t.Errorf("status = %q, want pending", s.Status)
	}
	if !s.RecurrenceID.Valid || s.RecurrenceID.String != rec.ID {
		t.Errorf("recurrence_id = %+v, want %q", s.RecurrenceID, rec.ID)
	}
	if s.PublishAt != at.UnixMilli() {
		t.Errorf("publish_at = %d, want %d", s.PublishAt, at.UnixMilli())
	}
}

func TestCreateRecurrenceRequiresPositiveInterval(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	if _, err := CreateRecurrence(ctx, db, "foo.bar", `{}`, 0); err == nil {
		t.Fatal("CreateRecurrence with zero interval did not error")
	}
}

func TestCancelRecurrenceCancelsPendingChild(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	rec, err := CreateRecurrence(ctx, db, "foo.bar", `{"p":1}`, time.Minute)
	if err != nil {
		t.Fatalf("CreateRecurrence: %v", err)
	}
	occ, err := CreateScheduledOccurrence(ctx, db, rec.Subject, rec.Payload, time.Now().Add(time.Minute), rec.ID)
	if err != nil {
		t.Fatalf("CreateScheduledOccurrence: %v", err)
	}
	// A fired child from an earlier occurrence must stay fired; only the
	// pending next one is withdrawn.
	past, err := CreateScheduledOccurrence(ctx, db, rec.Subject, rec.Payload, time.Now().Add(-time.Minute), rec.ID)
	if err != nil {
		t.Fatalf("CreateScheduledOccurrence: %v", err)
	}
	if won, err := ClaimForFire(ctx, db, past.ID); err != nil || !won {
		t.Fatalf("ClaimForFire: won=%v err=%v", won, err)
	}

	won, err := CancelRecurrence(ctx, db, rec.ID)
	if err != nil {
		t.Fatalf("CancelRecurrence: %v", err)
	}
	if !won {
		t.Fatal("first cancel should win")
	}
	won, err = CancelRecurrence(ctx, db, rec.ID)
	if err != nil {
		t.Fatalf("CancelRecurrence: %v", err)
	}
	if won {
		t.Fatal("second cancel must lose")
	}

	var recStatus string
	if err := db.QueryRowContext(ctx,
		`SELECT status FROM recurrences WHERE id = ?`, rec.ID).Scan(&recStatus); err != nil {
		t.Fatalf("read recurrence: %v", err)
	}
	if recStatus != "cancelled" {
		t.Errorf("recurrence status = %q, want cancelled", recStatus)
	}
	// The pending next occurrence is cancelled; the fired one stays fired.
	var occStatus, pastStatus string
	if err := db.QueryRowContext(ctx,
		`SELECT status FROM scheduler WHERE id = ?`, occ.ID).Scan(&occStatus); err != nil {
		t.Fatalf("read occurrence: %v", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT status FROM scheduler WHERE id = ?`, past.ID).Scan(&pastStatus); err != nil {
		t.Fatalf("read past occurrence: %v", err)
	}
	if occStatus != "cancelled" {
		t.Errorf("pending occurrence status = %q, want cancelled", occStatus)
	}
	if pastStatus != "fired" {
		t.Errorf("fired occurrence status = %q, want fired", pastStatus)
	}
}

func TestListRecurrencesOrdersActiveFirst(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	early, err := CreateRecurrence(ctx, db, "a", `{"p":1}`, time.Minute)
	if err != nil {
		t.Fatalf("CreateRecurrence: %v", err)
	}
	cancelled, err := CreateRecurrence(ctx, db, "b", `{"p":2}`, 2*time.Minute)
	if err != nil {
		t.Fatalf("CreateRecurrence: %v", err)
	}
	late, err := CreateRecurrence(ctx, db, "c", `{"p":3}`, 3*time.Minute)
	if err != nil {
		t.Fatalf("CreateRecurrence: %v", err)
	}
	if won, err := CancelRecurrence(ctx, db, cancelled.ID); err != nil || !won {
		t.Fatalf("CancelRecurrence: won=%v err=%v", won, err)
	}

	got, err := ListRecurrences(ctx, db)
	if err != nil {
		t.Fatalf("ListRecurrences: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListRecurrences = %d rows, want 3", len(got))
	}
	if got[0].ID != early.ID || got[0].Status != "active" {
		t.Errorf("got[0] = %+v, want active %s", got[0], early.ID)
	}
	if got[1].ID != late.ID || got[1].Status != "active" {
		t.Errorf("got[1] = %+v, want active %s", got[1], late.ID)
	}
	if got[2].ID != cancelled.ID || got[2].Status != "cancelled" {
		t.Errorf("got[2] = %+v, want cancelled %s", got[2], cancelled.ID)
	}
}

func TestClaimAndAdvanceFiresAndAdvances(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	rec, err := CreateRecurrence(ctx, db, "foo.bar", `{"p":1}`, 5*time.Minute)
	if err != nil {
		t.Fatalf("CreateRecurrence: %v", err)
	}
	first, err := CreateScheduledOccurrence(ctx, db, rec.Subject, rec.Payload, time.UnixMilli(1_000_000), rec.ID)
	if err != nil {
		t.Fatalf("CreateScheduledOccurrence: %v", err)
	}

	won, next, err := ClaimAndAdvance(ctx, db, first.ID, "fired")
	if err != nil {
		t.Fatalf("ClaimAndAdvance: %v", err)
	}
	if !won {
		t.Fatal("first claim should win")
	}
	if next == nil {
		t.Fatal("no next occurrence scheduled")
	}
	if !next.RecurrenceID.Valid || next.RecurrenceID.String != rec.ID {
		t.Errorf("next recurrence_id = %+v, want %q", next.RecurrenceID, rec.ID)
	}
	if want := int64(1_000_000) + 5*time.Minute.Milliseconds(); next.PublishAt != want {
		t.Errorf("next publish_at = %d, want %d", next.PublishAt, want)
	}
	var status string
	if err := db.QueryRowContext(ctx,
		`SELECT status FROM scheduler WHERE id = ?`, first.ID).Scan(&status); err != nil {
		t.Fatalf("read first: %v", err)
	}
	if status != "fired" {
		t.Errorf("first status = %q, want fired", status)
	}

	// A second claim loses and must not schedule another occurrence.
	won, next, err = ClaimAndAdvance(ctx, db, first.ID, "fired")
	if err != nil {
		t.Fatalf("ClaimAndAdvance: %v", err)
	}
	if won {
		t.Fatal("second claim must lose")
	}
	if next != nil {
		t.Error("lost claim scheduled an occurrence")
	}
}

func TestClaimAndAdvanceExpiredAndAdvances(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	rec, err := CreateRecurrence(ctx, db, "foo.bar", `{"p":1}`, 10*time.Minute)
	if err != nil {
		t.Fatalf("CreateRecurrence: %v", err)
	}
	missed, err := CreateScheduledOccurrence(ctx, db, rec.Subject, rec.Payload, time.UnixMilli(2_000_000), rec.ID)
	if err != nil {
		t.Fatalf("CreateScheduledOccurrence: %v", err)
	}

	won, next, err := ClaimAndAdvance(ctx, db, missed.ID, "expired")
	if err != nil {
		t.Fatalf("ClaimAndAdvance: %v", err)
	}
	if !won {
		t.Fatal("first claim should win")
	}
	if next == nil {
		t.Fatal("skipped occurrence did not advance the schedule")
	}
	if want := int64(2_000_000) + 10*time.Minute.Milliseconds(); next.PublishAt != want {
		t.Errorf("next publish_at = %d, want %d", next.PublishAt, want)
	}
	var status string
	if err := db.QueryRowContext(ctx,
		`SELECT status FROM scheduler WHERE id = ?`, missed.ID).Scan(&status); err != nil {
		t.Fatalf("read missed: %v", err)
	}
	if status != "expired" {
		t.Errorf("missed status = %q, want expired", status)
	}
}

func TestClaimAndAdvanceSkipsOneShot(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	s, err := CreateScheduled(ctx, db, "foo.bar", `{}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	won, next, err := ClaimAndAdvance(ctx, db, s.ID, "fired")
	if err != nil {
		t.Fatalf("ClaimAndAdvance: %v", err)
	}
	if !won {
		t.Fatal("claim should win")
	}
	if next != nil {
		t.Error("one-shot claim scheduled an occurrence")
	}
}

func TestClaimAndAdvanceStopsWhenRecurrenceCancelled(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	rec, err := CreateRecurrence(ctx, db, "foo.bar", `{"p":1}`, time.Minute)
	if err != nil {
		t.Fatalf("CreateRecurrence: %v", err)
	}
	occ, err := CreateScheduledOccurrence(ctx, db, rec.Subject, rec.Payload, time.Now().Add(time.Minute), rec.ID)
	if err != nil {
		t.Fatalf("CreateScheduledOccurrence: %v", err)
	}
	// Cancel the parent directly (bypassing CancelRecurrence, which would also
	// withdraw the child) to prove the claim itself stops advancing.
	if _, err := db.ExecContext(ctx,
		`UPDATE recurrences SET status = 'cancelled' WHERE id = ?`, rec.ID); err != nil {
		t.Fatalf("cancel parent: %v", err)
	}

	won, next, err := ClaimAndAdvance(ctx, db, occ.ID, "fired")
	if err != nil {
		t.Fatalf("ClaimAndAdvance: %v", err)
	}
	if !won {
		t.Fatal("claim should win")
	}
	if next != nil {
		t.Error("cancelled recurrence advanced to another occurrence")
	}
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM scheduler WHERE recurrence_id = ?`, rec.ID).Scan(&n); err != nil {
		t.Fatalf("count occurrences: %v", err)
	}
	if n != 1 {
		t.Errorf("occurrence count = %d, want 1 (no advance)", n)
	}
}

func TestListUnpublishedAfterClaim(t *testing.T) {
	db, ctx := newSchedulerTestDB(t)
	s, err := CreateScheduled(ctx, db, "foo.bar", `{}`, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	if _, err := ClaimForFire(ctx, db, s.ID); err != nil {
		t.Fatalf("ClaimForFire: %v", err)
	}
	got, err := ListUnpublished(ctx, db)
	if err != nil {
		t.Fatalf("ListUnpublished: %v", err)
	}
	if len(got) != 1 || got[0].ID != s.ID {
		t.Errorf("ListUnpublished = %+v, want the fired-but-unmarked row", got)
	}
}
