package testenv

import (
	"strings"
	"sync"
	"testing"
)

// resetDotEnvForTest re-arms the once so a test can reload .env from its own
// working directory, making the dotenv tests order-independent: a sibling test
// calling DotEnvValue earlier cannot silently pin which .env this test sees.
func resetDotEnvForTest() {
	loadOnce = sync.Once{}
	dotEnv = nil
}

func TestParseDotEnv(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want map[string]string
	}{
		{
			name: "blank and comment lines skipped",
			in:   "\n# comment\n   # indented comment\nKEY=value\n",
			want: map[string]string{"KEY": "value"},
		},
		{
			name: "malformed line without equals ignored",
			in:   "NOSEPARATOR\nKEY=value\n",
			want: map[string]string{"KEY": "value"},
		},
		{
			name: "export prefix stripped",
			in:   "export KEY=value\n",
			want: map[string]string{"KEY": "value"},
		},
		{
			name: "double and single quotes stripped",
			in:   "A=\"va lue\"\nB='x'\n",
			want: map[string]string{"A": "va lue", "B": "x"},
		},
		{
			name: "empty key ignored",
			in:   "=value\nKEY=value\n",
			want: map[string]string{"KEY": "value"},
		},
		{
			name: "value containing equals preserved past first",
			in:   "KEY=a=b=c\n",
			want: map[string]string{"KEY": "a=b=c"},
		},
		{
			name: "key and value whitespace trimmed",
			in:   "  KEY  =  value  \n",
			want: map[string]string{"KEY": "value"},
		},
		{
			name: "inline hash kept as part of value",
			in:   "KEY=value # note\n",
			want: map[string]string{"KEY": "value # note"},
		},
		{
			name: "empty input yields empty map",
			in:   "",
			want: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDotEnv(strings.NewReader(tc.in))
			if len(got) != len(tc.want) {
				t.Fatalf("parseDotEnv: got %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Fatalf("parseDotEnv: key %q = %q, want %q (got %v)", k, got[k], v, got)
				}
			}
		})
	}
}

func TestFindDotEnvPath_NoEnvFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if path, ok := findDotEnvPath(); ok {
		t.Fatalf("findDotEnvPath = %q, want not-found", path)
	}
}

func TestDotEnvValue_NoEnvFileReturnsEmpty(t *testing.T) {
	resetDotEnvForTest()
	t.Chdir(t.TempDir())

	if got := DotEnvValue("SOMETHING_ABSENT"); got != "" {
		t.Fatalf("DotEnvValue = %q, want empty", got)
	}
	if got := EnvOrDefault("SOMETHING_ABSENT", "fallback"); got != "fallback" {
		t.Fatalf("EnvOrDefault = %q, want fallback", got)
	}
}

func TestEnvOrDefault_TrimsWhitespacePaddedEnvOverride(t *testing.T) {
	t.Setenv("APP_PADDED", "  x  ")
	if got := EnvOrDefault("APP_PADDED", "fallback"); got != "x" {
		t.Fatalf("EnvOrDefault = %q, want %q", got, "x")
	}
}

func TestEnvOrDefault_RespectsResolutionOrder(t *testing.T) {
	resetDotEnvForTest()
	t.Setenv("APP_ORDER", "  from-env  ")
	t.Chdir(t.TempDir())
	if got := EnvOrDefault("APP_ORDER", "fallback"); got != "from-env" {
		t.Fatalf("EnvOrDefault = %q, want trimmed env value", got)
	}
}
