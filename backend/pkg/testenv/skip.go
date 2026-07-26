package testenv

import (
	"os"
	"testing"
)

const (
	// EnvRunDBTests is the environment variable that enables database-backed tests.
	EnvRunDBTests = "RUN_DB_TESTS"
	// EnvRunNetworkTest is the environment variable that enables network tests.
	EnvRunNetworkTest = "RUN_NETWORK_TESTS"
	// EnvRunE2ETests is the environment variable that enables end-to-end tests.
	EnvRunE2ETests = "RUN_E2E_TESTS"
)

// SkipIfDBTestsDisabled skips the test if DB tests are not enabled.
func SkipIfDBTestsDisabled(tb testing.TB) {
	tb.Helper()
	if os.Getenv(EnvRunDBTests) == "" {
		tb.Skip("skipping db-backed test; set " + EnvRunDBTests + "=1 to run")
	}
}

// SkipIfNetworkTestsDisabled skips the test if network tests are not enabled.
func SkipIfNetworkTestsDisabled(tb testing.TB) {
	tb.Helper()
	if os.Getenv(EnvRunNetworkTest) == "" {
		tb.Skip("skipping network test; set " + EnvRunNetworkTest + "=1 to run")
	}
}

// SkipIfE2ETestsDisabled skips the test if e2e tests are not enabled.
func SkipIfE2ETestsDisabled(tb testing.TB) {
	tb.Helper()
	if os.Getenv(EnvRunE2ETests) == "" {
		tb.Skip("skipping e2e test; set " + EnvRunE2ETests + "=1 to run")
	}
}
