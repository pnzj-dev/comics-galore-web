package view

import (
	"comics-galore-web/internal/blog"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"log/slog"
)

type Handler interface {
	RenderIndexPage(c fiber.Ctx) error
	RenderPostPage(c fiber.Ctx) error
	RegisterRoutes(app *fiber.App)
}

type handler struct {
	svc    blog.Service
	logger *slog.Logger
}

func NewHandler(svc blog.Service, logger *slog.Logger) Handler {
	return &handler{
		svc:    svc,
		logger: logger,
	}
}

func (h handler) RenderIndexPage(c fiber.Ctx) error {
	//TODO implement me
	panic("implement me")
}

func (h handler) RenderPostPage(c fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))

	// 1. Fetch the data
	post, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		// Return a dedicated 404 HTML page instead of JSON
		return c.Status(404).Render("errors/404", nil)
	}

	// 2. Render the HTML template
	return c.Render("post_detail", fiber.Map{"post": post})
}

func (h handler) RegisterRoutes(app *fiber.App) {
	//TODO implement me
	panic("implement me")
}
