package search

import "testing"

func TestValidate(t *testing.T) {
	if err := (Input{}).Validate(); err == nil {
		t.Fatal("expected query error")
	}
	if err := (Input{Query: "term", Limit: new(101)}).Validate(); err == nil {
		t.Fatal("expected limit error")
	}
}
