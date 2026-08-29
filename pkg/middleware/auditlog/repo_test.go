package auditlog

import "testing"

func TestToJSONBCompatibleStringRemovesNUL(t *testing.T) {
	got := toJSONBCompatibleString([]byte(`{"message":"before\u0000after"}`))
	if got == nil || *got != `{"message":"beforeafter"}` {
		t.Fatalf("converted JSON = %v", got)
	}
}

func TestToJSONBCompatibleStringRemovesNULFromNonJSONBody(t *testing.T) {
	got := toJSONBCompatibleString([]byte("before\x00after"))
	if got == nil || *got != `"beforeafter"` {
		t.Fatalf("converted JSON = %v", got)
	}
}
