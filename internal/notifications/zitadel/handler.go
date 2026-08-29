// Package zitadel handles private HTTP notification-provider callbacks from Zitadel.
package zitadel

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	mail "github.com/wneessen/go-mail"

	"github.com/mhmdkzr/app/internal/config"
)

// Handler receives JSON notifications from Zitadel's HTTP provider.
type Handler struct {
	pathSecret string
	mailer     *mail.Client
	smtp       config.SMTPConfig
}

// NewHandler constructs the private notification handler.
func NewHandler(pathSecret string, mailer *mail.Client, smtp config.SMTPConfig) Handler {
	return Handler{pathSecret: pathSecret, mailer: mailer, smtp: smtp}
}

// Register registers the shared-secret route on the internal-only mux.
func (h Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks/zitadel/notifications/"+h.pathSecret, h.handle)
}

type notification struct {
	ContextInfo struct {
		RecipientEmailAddress string `json:"recipientEmailAddress"`
	} `json:"contextInfo"`
	TemplateData struct {
		Subject string `json:"subject"`
		Text    string `json:"text"`
	} `json:"templateData"`
}

func (h Handler) handle(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(r.URL.Path, "/webhooks/zitadel/notifications/")), []byte(h.pathSecret)) != 1 {
		http.NotFound(w, r)
		return
	}
	var payload notification
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(w, "invalid notification payload", http.StatusBadRequest)
		return
	}
	if payload.ContextInfo.RecipientEmailAddress == "" || payload.TemplateData.Subject == "" || payload.TemplateData.Text == "" {
		http.Error(w, "incomplete notification payload", http.StatusBadRequest)
		return
	}
	message := mail.NewMsg()
	if err := message.FromFormat(h.smtp.FromName, h.smtp.From); err != nil {
		slog.Error("build Zitadel notification sender", "error", err)
		http.Error(w, "notification delivery failed", http.StatusInternalServerError)
		return
	}
	if err := message.To(payload.ContextInfo.RecipientEmailAddress); err != nil {
		http.Error(w, "invalid notification recipient", http.StatusBadRequest)
		return
	}
	message.Subject(payload.TemplateData.Subject)
	message.SetBodyString(mail.TypeTextPlain, payload.TemplateData.Text)
	if err := h.mailer.DialAndSendWithContext(r.Context(), message); err != nil {
		slog.Error("deliver Zitadel notification", "error", err)
		http.Error(w, "notification delivery failed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
