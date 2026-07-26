package pagination

import "testing"

func TestNormalize(t *testing.T) {
	got := Normalize(Meta{})
	if got.Page != DefaultPage || got.Size != DefaultSize {
		t.Fatalf("defaults mismatch: %#v", got)
	}

	got = Normalize(Meta{Page: 3, Size: 25})
	if got.Page != 3 || got.Size != 25 {
		t.Fatalf("preserve values mismatch: %#v", got)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		meta    Meta
		wantErr bool
	}{
		{name: "valid", meta: Meta{Page: 1, Size: 10}},
		{name: "bad_page", meta: Meta{Page: 0, Size: 10}, wantErr: true},
		{name: "bad_size", meta: Meta{Page: 1, Size: 0}, wantErr: true},
		{name: "too_large", meta: Meta{Page: 1, Size: MaxPageSize + 1}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.meta)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
		})
	}
}
