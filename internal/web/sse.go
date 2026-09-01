package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/pipeline"
	"github.com/mhmdkzr/taskman/internal/task"
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
	for _, subject := range []string{"agent.>", "scheduler.>", events.CommandRunSubject} {
		unsub, err := s.cfg.Pub.SubscribeCore(subject, func(subject string, data []byte) {
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

// serveCommands runs the command consumer until ctx is cancelled: it listens
// for RunCommand requests on the core command subject and executes each task
// through the pipeline. It returns a stop func that unsubscribes and blocks
// until the consumer has shut down.
func (s *Server) serveCommands(ctx context.Context) (func(), error) {
	stop, err := s.cfg.Pub.SubscribeCore(events.CommandRunSubject, func(_ string, data []byte) {
		var cmd events.RunCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			slog.Error("web: bad run command", "error", err)
			return
		}
		for _, id := range cmd.TaskIDs {
			go s.runCommandTask(ctx, id)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe commands: %w", err)
	}
	return func() {
		stop()
	}, nil
}

// runCommandTask executes one task id from a RunCommand. Each command task
// runs in its own goroutine and its own worktree (see pipeline.RunTask), so
// concurrent commands run concurrently.
func (s *Server) runCommandTask(ctx context.Context, rawID string) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		slog.Error("web: invalid task id in command", "id", rawID, "error", err)
		return
	}
	t, err := task.GetTask(ctx, s.cfg.Store.RW(), id)
	if err != nil {
		slog.Error("web: get task for command", "task_id", id, "error", err)
		return
	}
	start := time.Now()
	if _, err := pipeline.RunTask(ctx, s.cfg.Pipeline, *t); err != nil {
		slog.Error("web: command run failed", "task_id", id, "error", err, "elapsed", time.Since(start))
		return
	}
	slog.Info("web: command run finished", "task_id", id, "elapsed", time.Since(start))
}
