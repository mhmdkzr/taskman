package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/web/components"
	"github.com/mhmdkzr/loop/pkg/jsonresp"
	"github.com/starfederation/datastar-go/datastar"
)

func index(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		view, err := tasksView(r.Context(), a)
		if err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
			return
		}
		if r.URL.Query().Has(datastar.DatastarKey) {
			if err := datastar.NewSSE(w, r).PatchElementTempl(components.App(view)); err != nil {
				slog.Error("patch app view", "error", err)
			}
			return
		}
		if err := components.App(view).Render(r.Context(), w); err != nil {
			slog.Error("render tasks page", "error", err)
		}
	}
}

func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, sessions.ErrSessionNotFound), errors.Is(err, task.ErrTaskNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
