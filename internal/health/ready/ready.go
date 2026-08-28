package ready

import (
	"context"
	"database/sql"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	tigerbeetle "github.com/tigerbeetle/tigerbeetle-go"
	mail "github.com/wneessen/go-mail"
	zitadelclient "github.com/zitadel/zitadel-go/v3/pkg/client"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/auth"
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
	Database    *CheckResult `json:"database,omitempty"`
	NATS        *CheckResult `json:"nats,omitempty"`
	JetStream   *CheckResult `json:"jetstream,omitempty"`
	Temporal    *CheckResult `json:"temporal,omitempty"`
	TigerBeetle *CheckResult `json:"tigerbeetle,omitempty"`
	Zitadel     *CheckResult `json:"zitadel,omitempty"`
	Mailer      *CheckResult `json:"mailer,omitempty"`
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
	c.JetStream = jetStreamCheck(ctx, a.Deps.JS)
	c.Temporal = temporalCheck(ctx, a.Deps.Temporal)
	c.TigerBeetle = tigerBeetleCheck(a.Deps.TigerBeetle)
	c.Zitadel = zitadelCheck(ctx, a.Deps.Zitadel)
	c.Mailer = mailerCheck(ctx, a.Deps.Mailer)

	overall := "ok"
	for _, r := range [...]*CheckResult{
		c.Database, c.NATS, c.JetStream, c.Temporal, c.TigerBeetle, c.Zitadel, c.Mailer,
	} {
		if r != nil && r.Status != "ok" {
			overall = "degraded"
			break
		}
	}

	return Response{Status: overall, Checks: c}
}

func tigerBeetleCheck(client tigerbeetle.Client) *CheckResult {
	if client == nil {
		return checkFail("not connected")
	}
	if err := client.Nop(); err != nil {
		return checkFail(err.Error())
	}
	return checkOK()
}

func zitadelCheck(ctx context.Context, client *zitadelclient.Client) *CheckResult {
	if client == nil {
		return checkFail("not connected")
	}
	cctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	if _, err := client.AuthService().Healthz(cctx, &auth.HealthzRequest{}); err != nil {
		return checkFail(err.Error())
	}
	return checkOK()
}

func mailerCheck(ctx context.Context, client *mail.Client) *CheckResult {
	if client == nil {
		return checkFail("not connected")
	}
	cctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	smtpClient, err := client.DialToSMTPClientWithContext(cctx)
	if err != nil {
		return checkFail(err.Error())
	}
	if err := client.CloseWithSMTPClient(smtpClient); err != nil {
		return checkFail(err.Error())
	}
	return checkOK()
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

func checkOK() *CheckResult {
	return &CheckResult{Status: "ok"}
}

func checkFail(reason string) *CheckResult {
	return &CheckResult{Status: "error", Error: reason}
}
