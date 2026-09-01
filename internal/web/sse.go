package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// sseKeepAlive is how often the SSE stream sends a comment line to keep the
// connection alive through proxies that idle-close.
const sseKeepAlive = 20 * time.Second

// handleEvents streams live bus events to a browser over Server-Sent Events.
// Each event is a JSON envelope {"subject": "...", "payload": {...}} where
// payload is the event's own JSON. Subscriptions are transient core NATS
// subscriptions created per connection and torn down when the client leaves.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	msgs := make(chan sseMsg, 256)
	var unsubs []func()
	for _, subject := range []string{"agent.>", "scheduler.>"} {
		unsub, err := s.pub.SubscribeCore(subject, func(subject string, data []byte) {
			select {
			case msgs <- sseMsg{subject: subject, data: data}:
			default:
				slog.Warn("web: sse client too slow, dropping event", "subject", subject)
			}
		})
		if err != nil {
			for _, u := range unsubs {
				u()
			}
			http.Error(w, fmt.Sprintf("subscribe %s: %v", subject, err), http.StatusInternalServerError)
			return
		}
		unsubs = append(unsubs, unsub)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	// Flush the response headers right away so the browser opens the stream
	// immediately instead of waiting for the first keepalive tick (which can
	// take 20s). Without this the EventSource sits in "connecting" and some
	// clients time out as stalled and reconnect in a loop.
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(sseKeepAlive)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case m := <-msgs:
			envelope, err := json.Marshal(struct {
				Subject string          `json:"subject"`
				Payload json.RawMessage `json:"payload"`
			}{Subject: m.subject, Payload: m.data})
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", envelope); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// sseMsg is one bus message ready to forward to an SSE client.
type sseMsg struct {
	subject string
	data    []byte
}
