package register

import (
	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/frontend/serve"
)

func RegisterRoutes(a app.App) {
	a.RegisterRoutes(serve.NewHandler().Route())
}
