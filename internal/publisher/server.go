package publisher

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	// StreamName is the JetStream stream that captures every message on the
	// bus and deduplicates by MsgID within dedupWindow.
	StreamName = "SCION"

	// dedupWindow is how long the stream remembers published dedup ids, so a
	// re-published message within that window is dropped as a duplicate.
	dedupWindow = 24 * time.Hour

	// connectHost is the interface the embedded NATS server binds.
	connectHost = "127.0.0.1"
)

// ConnectOn starts an embedded NATS server on the given port and returns a
// Publisher for the bus. A non-positive port picks a random free port. The
// embedded server exists only for tests; production runs against an external
// NATS server via ConnectURL.
func ConnectOn(port int) (Publisher, error) {
	if port <= 0 {
		port = -1 // RANDOM_PORT: let the server pick a free port.
	}
	return connect(port)
}

// ConnectURL connects to an external NATS server at url — which must be
// running with JetStream enabled — and returns a Publisher for the bus. The
// SCION stream is added on connect, idempotently: a reconnect or restart
// reuses the existing stream as-is.
func ConnectURL(url string) (Publisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return Publisher{}, fmt.Errorf("connect to %s: %w", url, err)
	}
	if err := ensureStream(nc); err != nil {
		nc.Close()
		return Publisher{}, fmt.Errorf("add stream on %s: %w (is JetStream enabled on the server?)", url, err)
	}
	return NewPublisher(nc), nil
}

func connect(port int) (Publisher, error) {
	if port > 0 {
		if err := checkPortFree(port); err != nil {
			return Publisher{}, err
		}
	}

	opts := &server.Options{
		ServerName: "taskman",
		Host:       connectHost,
		Port:       port,
		JetStream:  true,
	}

	natsServer, err := server.NewServer(opts)
	if err != nil {
		return Publisher{}, fmt.Errorf("failed to create NATS server: %w", err)
	}

	go natsServer.Start()

	if !natsServer.ReadyForConnections(1 * time.Second) {
		return Publisher{}, fmt.Errorf("NATS server failed to start on %s:%d (check the server logs)", connectHost, port)
	}

	nc, err := nats.Connect(natsServer.ClientURL())
	if err != nil {
		return Publisher{}, fmt.Errorf("failed to connect to the NATS server: %w", err)
	}

	if err := ensureStream(nc); err != nil {
		return Publisher{}, err
	}

	return NewPublisher(nc), nil
}

// checkPortFree verifies nothing is bound to the port yet, so a busy port
// surfaces as an obvious error rather than a server that never becomes ready.
func checkPortFree(port int) error {
	addr := net.JoinHostPort(connectHost, strconv.Itoa(port))
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is not available: %w (is another process using %s?)", port, err, addr)
	}
	return l.Close()
}

// ensureStream creates the SCION stream if it does not already exist.
// CreateOrUpdateStream is idempotent, so a restart leaves it untouched.
func ensureStream(nc *nats.Conn) error {
	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       StreamName,
		Subjects:   []string{"agent.>", "scheduler.>"},
		Storage:    jetstream.MemoryStorage,
		Duplicates: dedupWindow,
	})
	if err != nil {
		return fmt.Errorf("failed to add stream %s: %w", StreamName, err)
	}
	return nil
}
