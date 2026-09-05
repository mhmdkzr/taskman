// Package datetime implements timestamp conversion.
package datetime

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type output struct {
	Datetime         string `json:"datetime"`
	Timezone         string `json:"timezone"`
	UnixSeconds      int64  `json:"unix_seconds"`
	UnixMilliseconds int64  `json:"unix_milliseconds"`
	UnixMicroseconds int64  `json:"unix_microseconds"`
	UnixNanoseconds  int64  `json:"unix_nanoseconds"`
}

func execute(_ context.Context, in input) (output, error) {
	return executeAt(in, time.Now)
}

func executeAt(in input, currentTime func() time.Time) (output, error) {
	zone := strings.TrimSpace(in.Timezone)
	if zone == "" {
		zone = "UTC"
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return output{}, fmt.Errorf("datetime: invalid timezone %q: %w", zone, err)
	}

	instant := currentTime()
	if in.Timestamp != nil {
		instant, err = timestampTime(*in.Timestamp, in.Unit)
		if err != nil {
			return output{}, fmt.Errorf("datetime: %w", err)
		}
	}
	instant = instant.In(location)

	return output{
		Datetime: instant.Format(time.RFC3339Nano), Timezone: zone,
		UnixSeconds: instant.Unix(), UnixMilliseconds: instant.UnixMilli(),
		UnixMicroseconds: instant.UnixMicro(), UnixNanoseconds: instant.UnixNano(),
	}, nil
}

func timestampTime(timestamp int64, unit string) (time.Time, error) {
	unit = strings.ToLower(strings.TrimSpace(unit))
	if unit == "" {
		unit = inferUnit(timestamp)
	}

	switch unit {
	case "s", "sec", "second", "seconds":
		return time.Unix(timestamp, 0), nil
	case "ms", "msec", "millisecond", "milliseconds":
		return time.Unix(timestamp/1e3, (timestamp%1e3)*1e6), nil
	case "us", "usec", "microsecond", "microseconds":
		return time.Unix(timestamp/1e6, (timestamp%1e6)*1e3), nil
	case "ns", "nsec", "nanosecond", "nanoseconds":
		return time.Unix(0, timestamp), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp unit %q; use s, ms, us, or ns", unit)
	}
}

func inferUnit(timestamp int64) string {
	value := timestamp
	if value < 0 {
		value = -(value + 1)
		value++
	}
	switch {
	case value >= 1e18:
		return "ns"
	case value >= 1e15:
		return "us"
	case value >= 1e12:
		return "ms"
	default:
		return "s"
	}
}
