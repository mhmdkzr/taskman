package list

import "testing"

func TestValidateLimit(t *testing.T) {
	if err := (Input{Limit: new(101)}).Validate(); err == nil {
		t.Fatal("expected limit error")
	}
}
