package specificationreject

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func TestReject(t *testing.T) {
	dir := t.TempDir()
	value := task.Task{
		ID:            "abc",
		State:         task.StateSpecificationReview,
		Definition:    "def",
		Specification: "spec",
		DoneWhen:      "done",
	}
	if err := store.Write(dir, value); err != nil {
		t.Fatalf("write task: %v", err)
	}

	got, err := Reject(dir, Request{ID: "abc", Reason: "narrow the scope"})
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if got.State != task.StateSpecify {
		t.Fatalf("state = %q, want specify", got.State)
	}
	if len(got.SpecificationReviews) != 1 || got.SpecificationReviews[0].Approved {
		t.Fatalf("specification_reviews = %#v, want one rejection", got.SpecificationReviews)
	}
	if got.SpecificationReviews[0].Comment != "narrow the scope" {
		t.Fatalf("comment = %q, want rejection reason", got.SpecificationReviews[0].Comment)
	}
}

func TestRejectRequiresReason(t *testing.T) {
	if _, err := Reject(t.TempDir(), Request{ID: "abc"}); err == nil {
		t.Fatal("Reject without reason: want error")
	}
}
