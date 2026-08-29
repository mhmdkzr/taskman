package identity

import "testing"

func TestNewUserIDGeneratesUUIDv7(t *testing.T) {
	t.Parallel()
	id, err := NewUserID()
	if err != nil {
		t.Fatal(err)
	}
	if id.UUID().Version() != 7 {
		t.Fatalf("UUID version = %d, want 7", id.UUID().Version())
	}
}

func TestNewZitadelSubject(t *testing.T) {
	t.Parallel()
	if _, err := NewZitadelSubject(" "); err == nil {
		t.Fatal("empty subject accepted")
	}
	if got, err := NewZitadelSubject(" subject "); err != nil || got != "subject" {
		t.Fatalf("got %q, %v", got, err)
	}
}
