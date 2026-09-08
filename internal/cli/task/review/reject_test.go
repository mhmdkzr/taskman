package review

import "testing"

func TestReject(t *testing.T) {
	dir, id := newTestRepo(t)
	if err := runCmd(t, dir, Record(), id, "--approved=true"); err != nil {
		t.Fatalf("record: %v", err)
	}

	if err := runCmd(t, dir, Reject(), id, "--reason", "please add tests"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	got := getTask(t, dir, id)
	if got.Status.Review.State != "in_progress" {
		t.Fatalf("review.state = %v, want in_progress", got.Status.Review.State)
	}
	if len(got.HumanReviews) != 1 || got.HumanReviews[0].Approved || got.HumanReviews[0].Comment != "please add tests" {
		t.Fatalf("human reviews = %+v", got.HumanReviews)
	}
}

func TestRejectRequiresReason(t *testing.T) {
	dir, id := newTestRepo(t)
	if err := runCmd(t, dir, Record(), id, "--approved=true"); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := runCmd(t, dir, Reject(), id); err == nil {
		t.Fatal("reject without --reason: want error, got nil")
	}
}

func TestRejectFailsWhenNotPending(t *testing.T) {
	dir, id := newTestRepo(t)
	if err := runCmd(t, dir, Record(), id, "--approved=true"); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := runCmd(t, dir, Reject(), id, "--reason", "first"); err != nil {
		t.Fatalf("first reject: %v", err)
	}
	if err := runCmd(t, dir, Reject(), id, "--reason", "second"); err == nil {
		t.Fatal("reject a review already in_progress: want error, got nil")
	}
}
