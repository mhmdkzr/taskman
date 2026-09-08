package review

import "testing"

func TestApprove(t *testing.T) {
	dir, id := newTestRepo(t)
	if err := runCmd(t, dir, Record(), id, "--approved=true"); err != nil {
		t.Fatalf("record: %v", err)
	}

	if err := runCmd(t, dir, Approve(), id, "--comment", "LGTM"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	got := getTask(t, dir, id)
	if got.Status.Review.State != "done" {
		t.Fatalf("review.state = %v, want done", got.Status.Review.State)
	}
	if len(got.HumanReviews) != 1 || !got.HumanReviews[0].Approved || got.HumanReviews[0].Comment != "LGTM" {
		t.Fatalf("human reviews = %+v", got.HumanReviews)
	}
}

func TestApproveFailsWhenNotPending(t *testing.T) {
	dir, id := newTestRepo(t)
	if err := runCmd(t, dir, Record(), id, "--approved=true"); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := runCmd(t, dir, Approve(), id); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	if err := runCmd(t, dir, Approve(), id); err == nil {
		t.Fatal("approve an already-approved review: want error, got nil")
	}
}
