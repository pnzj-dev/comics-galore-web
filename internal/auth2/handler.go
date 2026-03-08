package auth2

import (
	"comics-galore-web/cmd/web/auth2"
	"comics-galore-web/internal/config"
	"github.com/gofiber/fiber/v3"
)

type Handler interface {
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
	app.Get("/auth/modal", h.showAuthModal)
	app.Get("/auth/modal/tab", h.loadAuthTab)
}

func (h *handler) showAuthModal(c fiber.Ctx) error {
	vm := auth2.AuthModalVM{
		Tab:              c.Query("tab", "login"),
		Errors:           make(map[string]string),
		Values:           make(map[string]string),
		TurnstileSiteKey: h.cfg.Get().Cloudflare.TurnstileSiteKey,
	}
	return auth2.AuthModal(vm).Render(c.Context(), c.Response().BodyWriter())
}

func (h *handler) loadAuthTab(c fiber.Ctx) error {
	vm := auth2.AuthModalVM{
		Tab:              c.Query("tab", "login"),
		Errors:           make(map[string]string),
		Values:           make(map[string]string),
		TurnstileSiteKey: h.cfg.Get().Cloudflare.TurnstileSiteKey,
	}
	return auth2.AuthTabContent(vm).Render(c.Context(), c.Response().BodyWriter())
}
