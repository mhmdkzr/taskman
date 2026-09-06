package delete

import (
	"testing"
	"uuid"
)

func TestInputValidate(t *testing.T) {
	if err := (Input{ID: uuid.NewV7().String()}).Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	if err := (Input{ID: "bad"}).Validate(); err == nil {
		t.Fatal("Validate() returned nil for invalid ID")
	}
}
