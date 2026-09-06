package delete

import "testing"

func TestValidate(t *testing.T) {
	if err := (Input{}).Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
