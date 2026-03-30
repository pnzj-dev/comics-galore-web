package modal

import (
	"comics-galore-web/cmd/web/handlers/view"
	"comics-galore-web/cmd/web/views/partials/modals"
	"comics-galore-web/internal/auth"
	"comics-galore-web/internal/config"
	"github.com/a-h/templ"
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
	authGroup := app.Group("/auth/modal", auth.SessionLoader(h.cfg), view.GetAppContext(h.cfg.Get()))
	authGroup.Get("/login", func(c fiber.Ctx) error { return h.render(c, modals.Signin(make(map[string]string))) })
	authGroup.Get("/signup", func(c fiber.Ctx) error { return h.render(c, modals.Signup(make(map[string]string))) })
	authGroup.Get("/forgot", func(c fiber.Ctx) error { return h.render(c, modals.ResetPassword(make(map[string]string))) })

	userGroup := app.Group("/user/menu", auth.SessionLoader(h.cfg), view.GetAppContext(h.cfg.Get()))
	userGroup.Get("/upload", func(c fiber.Ctx) error { return h.render(c, modals.Upload()) })
	userGroup.Get("/signout", func(c fiber.Ctx) error { return h.render(c, modals.Logout()) })
	userGroup.Get("/profile", func(c fiber.Ctx) error { return h.render(c, modals.Profile()) })
	userGroup.Get("/settings", func(c fiber.Ctx) error { return h.render(c, modals.Settings()) })
	userGroup.Get("/messages", func(c fiber.Ctx) error { return h.render(c, modals.DirectMessage()) })
}

// Helper to keep the route definitions readable
func (h *handler) render(c fiber.Ctx, component templ.Component) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return component.Render(c.Context(), c.Response().BodyWriter())
}
