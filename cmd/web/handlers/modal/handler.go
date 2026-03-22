package modal

import (
	"comics-galore-web/internal/config"
	"github.com/gofiber/fiber/v3"
)

type Handler interface {
	RegisterRoutes(app *fiber.App)
}

type handler struct {
	cfg config.Service
}

func NewHandler(cfg config.Service) Handler {
	return &handler{
		cfg: cfg,
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	group := app.Group("/auth/modal")
	group.Get("/login", h.showSigninModal)
	group.Get("/signup", h.showSignupModal)
	group.Get("/forgot", h.showForgotModal)
}
