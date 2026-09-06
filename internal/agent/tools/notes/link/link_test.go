package link

import "testing"

func TestValidate(t *testing.T) {
	if err := (Input{From: "a", To: "a"}).Validate(); err == nil {
		t.Fatal("expected self-link validation error")
	}
	if err := (Input{From: "a", To: "b"}).Validate(); err != nil {
		t.Fatal(err)
	}
}
