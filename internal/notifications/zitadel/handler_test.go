package zitadel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mhmdkzr/app/internal/config"
)

func TestPrivateRouteRejectsInvalidPayloadBeforeDelivery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	NewHandler("shared-secret", nil, config.SMTPConfig{}).Register(mux)
	wrong := httptest.NewRecorder()
	mux.ServeHTTP(wrong, httptest.NewRequest(http.MethodPost, "/webhooks/zitadel/notifications/not-the-secret", nil))
	if wrong.Code != http.StatusNotFound {
		t.Fatalf("wrong secret status = %d, want 404", wrong.Code)
	}
	bad := httptest.NewRecorder()
	mux.ServeHTTP(
		bad,
		httptest.NewRequest(http.MethodPost, "/webhooks/zitadel/notifications/shared-secret", strings.NewReader("{}")),
	)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid payload status = %d, want 400", bad.Code)
	}
}
