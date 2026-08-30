package loginui

import (
	"context"
	"fmt"

	mail "github.com/wneessen/go-mail"
)

// Mailer sends the transactional emails this slice must hand-deliver itself
// (registration verification, password reset) rather than relying on
// ZITADEL's own notification pipeline — see registration.go. It is an
// interface so tests can substitute a recorder instead of a real SMTP
// client.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// smtpMailer sends via the app's own configured SMTP client (the same one
// used elsewhere in the app, e.g. internal/notifications/zitadel).
type smtpMailer struct {
	client   *mail.Client
	from     string
	fromName string
}

// NewSMTPMailer wraps an existing SMTP client as a Mailer.
func NewSMTPMailer(client *mail.Client, from, fromName string) Mailer {
	return &smtpMailer{client: client, from: from, fromName: fromName}
}

func (m *smtpMailer) Send(ctx context.Context, to, subject, body string) error {
	message := mail.NewMsg()
	if err := message.FromFormat(m.fromName, m.from); err != nil {
		return fmt.Errorf("set sender: %w", err)
	}
	if err := message.To(to); err != nil {
		return fmt.Errorf("set recipient: %w", err)
	}
	message.Subject(subject)
	// Bodies here are plain ASCII (links, short instructions). go-mail only
	// skips quoted-printable's soft line-wrapping for NoEncoding — anything
	// else, including EncodingUSASCII, still goes through a
	// quotedprintable.Writer — so use NoEncoding to keep URLs intact for
	// anything parsing the raw message (e.g. a mail-catcher's API in tests).
	message.SetBodyString(mail.TypeTextPlain, body, mail.WithPartEncoding(mail.NoEncoding))
	if err := m.client.DialAndSendWithContext(ctx, message); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}
