package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/scheduler"
	"github.com/mhmdkzr/taskman/internal/store"
)

func newSchedForTest(t *testing.T) (*sql.DB, *scheduler.Scheduler, publisher.Publisher) {
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
		t.Fatalf("publisher.Connect: %v", err)
	}
	return db, scheduler.New(db, pub, 5*time.Second), pub
}

func TestRunRequestSubject(t *testing.T) {
	if got := (RunRequest{}).Subject(); got != RunSubject {
		t.Errorf("Subject() = %q, want %q", got, RunSubject)
	}
}

func TestScheduleRequiresPublishTime(t *testing.T) {
	_, sched, _ := newSchedForTest(t)
	if _, err := Schedule(context.Background(), sched, RunRequest{Prompt: "hi"}, time.Time{}); err == nil {
		t.Fatal("Schedule with zero time did not error")
	}
}

func TestScheduleFiresOnRunSubject(t *testing.T) {
	db, sched, pub := newSchedForTest(t)
	ctx := context.Background()

	got := make(chan publisher.Message, 1)
	sub, err := pub.Subscribe(ctx, RunSubject, "schedule-test", func(m publisher.Message) {
		got <- m
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Stop()

	id, err := Schedule(ctx, sched, RunRequest{Prompt: "hello", Model: "m"}, time.Now())
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if id == "" {
		t.Fatal("empty schedule id")
	}
	if err := sched.Poll(ctx); err != nil {
		t.Fatalf("poll: %v", err)
	}

	select {
	case m := <-got:
		var req RunRequest
		if err := json.Unmarshal(m.Data(), &req); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if req.Prompt != "hello" || req.Model != "m" {
			t.Errorf("RunRequest = %+v", req)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("message never published")
	}

	rows, err := store.ListUnpublished(ctx, db)
	if err != nil {
		t.Fatalf("ListUnpublished: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("message left unpublished: %v", rows)
	}
}

func TestMeaningfulOverride(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"default", ""},
		{"DEFAULT", ""},
		{"  default  ", ""},
		{"deepseek-v4-flash", "deepseek-v4-flash"},
		{"low", "low"},
	}
	for _, tc := range cases {
		if got := MeaningfulOverride(tc.in); got != tc.want {
			t.Errorf("MeaningfulOverride(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
