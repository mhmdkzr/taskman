# `pkg/natsembed`

Starts an in-process embedded NATS server with JetStream via `Connect()`, returning a connected `*nats.Conn` and `jetstream.JetStream` context. Used in tests and lightweight deployments.
