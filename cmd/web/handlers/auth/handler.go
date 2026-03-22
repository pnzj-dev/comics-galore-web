package auth

import (
	cloudflare2 "comics-galore-web/cmd/web/middlewares/cloudflare"
	"comics-galore-web/internal/cloudflare"
	"comics-galore-web/internal/config"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"log/slog"
)

type handler struct {
	cfg       config.Service
	logger    *slog.Logger
	turnstile cloudflare.Turnstile
	validate  *validator.Validate
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
}

func NewHandler(cfg config.Service, turnstile cloudflare.Turnstile) Handler {
	return &handler{
		cfg:       cfg,
		turnstile: turnstile,
		validate:  validator.New(),
		logger:    cfg.GetLogger().With("component", "auth_proxy"),
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	group := app.Group("/api/v1/auth")
	//group.Use(h.validateInput(), h.withCookieSync(), h.handleAuthResponse)
	group.Use(cloudflare2.VerifyTurnstile(h.turnstile, h.cfg, h.renderError), h.validateInput(), h.withCookieSync(), h.handleAuthResponse)
	group.All("/*", h.proxyToBetterAuth)
}
