package view

import (
	"comics-galore-web/cmd/web/auth"
	"comics-galore-web/cmd/web/details"
	"comics-galore-web/cmd/web/home"
	"comics-galore-web/cmd/web/menu"
	"comics-galore-web/cmd/web/templates"
	authHelper "comics-galore-web/internal/auth"
	"comics-galore-web/internal/blog"
	"comics-galore-web/internal/config"
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"log/slog"
	"strconv"
)

type Handler interface {
	RenderIndexPage(c fiber.Ctx) error
	RenderPostPage(c fiber.Ctx) error
	RegisterRoutes(app *fiber.App)
}

type handler struct {
	svc      blog.Service
	cfg      config.Service
	logger   *slog.Logger
	variants map[string]string
}

func NewHandler(cfg config.Service, svc blog.Service) Handler {
	return &handler{
		svc:      svc,
		cfg:      cfg,
		variants: map[string]string{"view": "public", "preview": "cover", "thumbnail": "thumbnail"},
		logger:   cfg.GetLogger().With("component", "view_handler"),
	}
}

func authAndRender(c fiber.Ctx, pageContent templ.Component) error {
	claims := authHelper.GetClaims(c)
	headerButton := auth.AuthModal()
	if claims != nil {
		headerButton = menu.AvatarMenu(claims)
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return templates.BaseLayout(headerButton, pageContent).Render(c.Context(), c.Response().BodyWriter())
}

func (h *handler) RenderPostPage(c fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	//slug := c.Params("slug")
	post, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return authAndRender(c, details.Details(post, []blog.Post{}, authHelper.GetClaims(c), h.variants))
}

func (h *handler) RenderIndexPage(c fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	posts, err := h.svc.List(c.Context(), int32(limit), int32(offset))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return authAndRender(c, home.HomePage(posts))
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	app.Get("/", authHelper.HasSession(h.cfg), h.RenderIndexPage)
	app.Get("/post/:id/:slug/", authHelper.HasSession(h.cfg), h.RenderPostPage)
}
