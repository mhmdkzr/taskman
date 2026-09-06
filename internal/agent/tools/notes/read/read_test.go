package read

import "testing"

func TestValidate(t *testing.T) {
	if err := (Input{}).Validate(); err == nil {
		t.Fatal("expected name error")
	}
}
