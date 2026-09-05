# Datetime Tool

The `datetime` tool returns the current date and time, or converts a supplied
Unix timestamp. It uses UTC when `timezone` is omitted and accepts IANA timezone
names such as `Europe/Berlin` or `America/New_York`.

The optional `timestamp` can be expressed in seconds, milliseconds,
microseconds, or nanoseconds. Set `timestamp_unit` to `s`, `ms`, `us`, or `ns`;
when omitted, the unit is inferred from the timestamp magnitude.

The result includes the RFC3339 timestamp with nanosecond precision and Unix
timestamps in all four supported units.
