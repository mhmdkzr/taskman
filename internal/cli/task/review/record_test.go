package review

import "testing"

func TestRecordApproved(t *testing.T) {
	dir, id := newTestRepo(t)

	if err := runCmd(t, dir, Record(), id, "--approved=true"); err != nil {
		t.Fatalf("record: %v", err)
	}
	got := getTask(t, dir, id)
	if got.Status.Verification.State != "done" {
		t.Fatalf("verification.state = %v, want done", got.Status.Verification.State)
	}
	if got.Status.Review.State != "pending" {
		t.Fatalf("review.state = %v, want pending", got.Status.Review.State)
	}
}

func TestRecordRejectedTwiceBlocks(t *testing.T) {
	dir, id := newTestRepo(t)

	if err := runCmd(t, dir, Record(), id, "--approved=false", "--finding", "x.go=bug"); err != nil {
		t.Fatalf("record 1: %v", err)
	}
	got := getTask(t, dir, id)
	if got.State == "blocked" {
		t.Fatal("first rejection must not block the task")
	}

	if err := runCmd(t, dir, Record(), id, "--approved=false", "--finding", "x.go=still buggy"); err != nil {
		t.Fatalf("record 2: %v", err)
	}
	got = getTask(t, dir, id)
	if got.State != "blocked" {
		t.Fatalf("state = %v, want blocked after second rejection", got.State)
	}
}

func TestRecordRequiresApprovedFlag(t *testing.T) {
	dir, id := newTestRepo(t)
	if err := runCmd(t, dir, Record(), id); err == nil {
		t.Fatal("record without --approved: want error, got nil")
	}
}
