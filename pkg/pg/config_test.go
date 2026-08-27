package pg

import "testing"

func TestConfigDSN(t *testing.T) {
	got := Config{
		Host:     "db",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Database: "app",
		SSLMode:  "disable",
	}.DSN()

	want := "postgres://user:pass@db:5432/app?sslmode=disable"
	if got != want {
		t.Fatalf("dsn mismatch: got=%q want=%q", got, want)
	}
}
