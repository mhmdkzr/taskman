package pg

import "testing"

func TestConfigDSN(t *testing.T) {
	got := Config{
		Host:     "db",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Database: "core",
		SSLMode:  "disable",
	}.DSN()

	want := "postgres://user:pass@db:5432/core?sslmode=disable"
	if got != want {
		t.Fatalf("dsn mismatch: got=%q want=%q", got, want)
	}
}
