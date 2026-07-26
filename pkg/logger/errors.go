package logger

import "errors"

var (
	errInvalidLogFormat    = errors.New("invalid log format")
	errInvalidNATSLogLevel = errors.New("invalid nats log level")
)
