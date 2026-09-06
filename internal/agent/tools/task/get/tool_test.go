package get

import "testing"

func TestInputValidate(t *testing.T) {
	if err := (Input{ID: "not-a-uuid"}).Validate(); err == nil {
		t.Fatal("Validate() returned nil for invalid ID")
	}
	if err := (Input{}).Validate(); err == nil {
		t.Fatal("Validate() returned nil for missing ID")
	}
}
