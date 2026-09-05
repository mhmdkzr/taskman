package testenv

import "testing"

func TestSkipHelpers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		envKey string
		skipFn func(testing.TB)
	}{
		{name: "db", envKey: "RUN_DB_TESTS", skipFn: SkipIfDBTestsDisabled},
		{name: "e2e", envKey: "RUN_E2E_TESTS", skipFn: SkipIfE2ETestsDisabled},
	} {
		for _, value := range []string{"", "0", "false"} {
			t.Run(tc.name+"_"+value+"_skips", func(t *testing.T) {
				t.Setenv(tc.envKey, value)
				tc.skipFn(t)
				t.Fatal("expected test to skip")
			})
		}
		t.Run(tc.name+"_set_does_not_skip", func(t *testing.T) {
			t.Setenv(tc.envKey, "1")
			tc.skipFn(t)
		})
	}
}
