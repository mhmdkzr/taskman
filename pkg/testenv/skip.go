package testenv

import (
	"os"
	"testing"
)

const (
	// EnvRunE2ETests is the environment variable that enables end-to-end tests.
	EnvRunE2ETests = "RUN_E2E_TESTS"
)

// SkipIfE2ETestsDisabled skips the test if e2e tests are not enabled. Only the
// explicit truthy value "1" enables; unset, empty, and non-truthy values (e.g.
// "0", "false") all skip.
func SkipIfE2ETestsDisabled(tb testing.TB) {
	tb.Helper()
	if os.Getenv(EnvRunE2ETests) != "1" {
		tb.Skip("skipping e2e test; set " + EnvRunE2ETests + "=1 to run")
	}
}
