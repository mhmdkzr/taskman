package specificationapprove

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func TestApprove(t *testing.T) {
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

	got, err := Approve(dir, Request{ID: "abc", Comment: "LGTM"})
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if got.State != task.StateImplement {
		t.Fatalf("state = %q, want implement", got.State)
	}
	if len(got.SpecificationReviews) != 1 || !got.SpecificationReviews[0].Approved {
		t.Fatalf("specification_reviews = %#v, want one approval", got.SpecificationReviews)
	}
}

func TestApproveRequiresPendingReview(t *testing.T) {
	dir := t.TempDir()
	value := task.Task{
		ID:            "abc",
		State:         task.StateImplement,
		Definition:    "def",
		Specification: "spec",
		DoneWhen:      "done",
	}
	if err := store.Write(dir, value); err != nil {
		t.Fatalf("write task: %v", err)
	}
	if _, err := Approve(dir, Request{ID: "abc"}); err == nil {
		t.Fatal("Approve outside specification_review: want error")
	}
}
