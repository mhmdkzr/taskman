package list

import "testing"

func TestValidateLimit(t *testing.T) {
	if err := (Input{Limit: ptr(101)}).Validate(); err == nil {
		t.Fatal("expected limit error")
	}
}
func ptr(v int) *int { return &v }
