package embeddednats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Connect starts an embedded NATS server and returns a connection and JetStream context.
func Connect() (*nats.Conn, jetstream.JetStream, error) {
	opts := &server.Options{
		ServerName: "embedded-nats-server",
		DontListen: true,
		JetStream:  true,
		StoreDir:   "",
	}

	natsServer, err := server.NewServer(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create NATS server: %w", err)
	}

	go natsServer.Start()

	if !natsServer.ReadyForConnections(1 * time.Second) {
		return nil, nil, fmt.Errorf("NATS server failed to start")
	}

	nc, err := nats.Connect("", nats.InProcessServer(natsServer))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to embedded NATS server: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return nc, js, nil
}
