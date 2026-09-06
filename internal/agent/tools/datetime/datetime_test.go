package datetime

import (
	"context"
	"testing"
	"time"
)

func TestExecuteDefaultsToUTC(t *testing.T) {
	result, err := executeAt(Input{}, func() time.Time {
		return time.Date(2026, time.September, 4, 12, 34, 56, 123456789, time.FixedZone("local", -7*60*60))
	})
	if err != nil {
		t.Fatal(err)
	}
	got := result
	if got.Datetime != "2026-09-04T19:34:56.123456789Z" || got.Timezone != "UTC" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestExecuteNanosecondTimestampAndTimezone(t *testing.T) {
	timestamp := int64(0)
	result, err := execute(context.Background(), Input{
		Timezone:  "America/New_York",
		Timestamp: &timestamp,
		Unit:      "ns",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := result
	if got.Datetime != "1969-12-31T19:00:00-05:00" || got.Timezone != "America/New_York" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestTimestampTimeInfersNanoseconds(t *testing.T) {
	const timestamp = int64(1_700_000_000_123_456_789)
	got, err := timestampTime(timestamp, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.UnixNano() != timestamp {
		t.Fatalf("UnixNano() = %d, want %d", got.UnixNano(), timestamp)
	}
}

func TestExecuteRejectsInvalidTimezoneAndUnit(t *testing.T) {
	for _, in := range []Input{{Timezone: "not/a_timezone"}, {Unit: "minutes", Timestamp: func() *int64 { v := int64(1); return &v }()}} {
		if _, err := execute(context.Background(), in); err == nil {
			t.Fatalf("execute(%+v) returned nil error", in)
		}
	}
}
