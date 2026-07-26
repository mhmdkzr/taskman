package ready

import (
	"context"
	"database/sql"
	"time"

	"github.com/nats-io/nats.go"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/mhmdkzr/app/internal/app"
)

// CheckResult is the status of a single dependency.
type CheckResult struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Checks holds the per-dependency readiness results.
type Checks struct {
	Database *CheckResult `json:"database,omitempty"`
	NATS     *CheckResult `json:"nats,omitempty"`
	Temporal *CheckResult `json:"temporal,omitempty"`
}

// Response is the readiness check result.
type Response struct {
	Status string `json:"status"`
	Checks Checks `json:"checks"`
}

const checkTimeout = 5 * time.Second

func readiness(ctx context.Context, a app.App) Response {
	var c Checks

	c.Database = dbCheck(ctx, a.Deps.DB)
	c.NATS = natsCheck(a.Deps.NC)
	c.Temporal = temporalCheck(ctx, a.Deps.Temporal)

	overall := "ok"
	for _, r := range [...]*CheckResult{
		c.Database, c.NATS, c.Temporal,
	} {
		if r != nil && r.Status != "ok" {
			overall = "degraded"
			break
		}
	}

	return Response{Status: overall, Checks: c}
}

func checkOK() *CheckResult {
	return &CheckResult{Status: "ok"}
}

func checkFail(reason string) *CheckResult {
	return &CheckResult{Status: "error", Error: reason}
}

func dbCheck(ctx context.Context, db *sql.DB) *CheckResult {
	if db == nil {
		return checkFail("not connected")
	}
	cctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	if err := db.PingContext(cctx); err != nil {
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

func temporalCheck(ctx context.Context, t temporalclient.Client) *CheckResult {
	if t == nil {
		return checkFail("not connected")
	}
	cctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	if _, err := t.CheckHealth(cctx, &temporalclient.CheckHealthRequest{}); err != nil {
		return checkFail(err.Error())
	}
	return checkOK()
}
