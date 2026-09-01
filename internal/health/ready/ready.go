package ready

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mhmdkzr/taskman/internal/app"
)

// CheckResult is the status of a single dependency.
type CheckResult struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Checks holds the per-dependency readiness results.
type Checks struct {
	NATS      *CheckResult `json:"nats,omitempty"`
	JetStream *CheckResult `json:"jetstream,omitempty"`
}

// Response is the readiness check result.
type Response struct {
	Status string `json:"status"`
	Checks Checks `json:"checks"`
}

const checkTimeout = 5 * time.Second

func readiness(ctx context.Context, a app.App) Response {
	var c Checks

	c.NATS = natsCheck(a.Deps.NC)
	c.JetStream = jetStreamCheck(ctx, a.Deps.JS)

	overall := "ok"
	for _, r := range [...]*CheckResult{
		c.NATS, c.JetStream,
	} {
		if r != nil && r.Status != "ok" {
			overall = "degraded"
			break
		}
	}

	return Response{Status: overall, Checks: c}
}

func jetStreamCheck(ctx context.Context, js jetstream.JetStream) *CheckResult {
	if js == nil {
		return checkFail("not connected")
	}
	cctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	if _, err := js.AccountInfo(cctx); err != nil {
		return checkFail(err.Error())
	}
	return checkOK()
}

func natsCheck(nc *nats.Conn) *CheckResult {
	if nc == nil || nc.Status() != nats.CONNECTED {
		return checkFail("not connected")
	}
	return checkOK()
}

func checkOK() *CheckResult {
	return &CheckResult{Status: "ok"}
}

func checkFail(reason string) *CheckResult {
	return &CheckResult{Status: "error", Error: reason}
}
