package task

import "testing"

func TestVerify(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	readyForVerify(t, dir, id)

	if _, err := runCmd(t, dir, Verify(), id, "--check", "vet=error", "--output", "boom"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	got := getJSON(t, dir, id)
	if len(got.Verifications) != 1 || got.Verifications[0].Passed() {
		t.Fatalf("verifications = %+v, want one failing entry", got.Verifications)
	}

	if _, err := runCmd(t, dir, Verify(), id, "--check", "vet=ok"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	got = getJSON(t, dir, id)
	if len(got.Verifications) != 2 || !got.Verifications[1].Passed() {
		t.Fatalf("verifications = %+v, want second entry passing", got.Verifications)
	}
}

func TestVerifyRequiresImplementation(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Verify(), id, "--check", "vet=ok"); err == nil {
		t.Fatal("verify before implement: want error, got nil")
	}
}

func TestVerifyInvalidCheckValue(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	readyForVerify(t, dir, id)
	if _, err := runCmd(t, dir, Verify(), id, "--check", "vet=maybe"); err == nil {
		t.Fatal("verify with invalid check value: want error, got nil")
	}
}
