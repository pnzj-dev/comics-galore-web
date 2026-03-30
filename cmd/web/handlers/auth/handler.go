package auth

import (
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

	group.Use(
		h.manageResponse, // Runs LAST (wraps everything)
		h.headerDebugMiddleware,
		h.createCookie,    // Runs SECOND TO LAST (wraps proxy)
		h.verifyTurnstile, // Runs immediately
		h.validateInput,   // Runs immediately
	)

	group.All("/*", h.proxyToBetterAuth)
}
