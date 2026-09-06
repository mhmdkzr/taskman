package edit

import "testing"

func TestValidate(t *testing.T) {
	for _, in := range []Input{{Body: "body"}, {Name: "name"}} {
		if err := in.Validate(); err == nil {
			t.Fatal("expected validation error")
		}
	}
}
