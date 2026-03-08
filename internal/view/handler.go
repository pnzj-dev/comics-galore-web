package view

import (
	"comics-galore-web/cmd/web/auth"
	"comics-galore-web/cmd/web/details"
	"comics-galore-web/cmd/web/header"
	"comics-galore-web/cmd/web/home"
	"comics-galore-web/cmd/web/menu"
	"comics-galore-web/cmd/web/templates"
	authHelper "comics-galore-web/internal/auth"
	"comics-galore-web/internal/blog"
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/database"
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"log/slog"
	"slices"
	"strconv"
	"strings"
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
	settings map[string][]string
}

func NewHandler(cfg config.Service, svc blog.Service) Handler {
	settings := make(map[string][]string)
	settings["Categories"] = []string{"big-tits", "giantess", "interracial"}
	return &handler{
		svc:      svc,
		cfg:      cfg,
		settings: settings,
		variants: map[string]string{"view": "public", "preview": "cover", "thumbnail": "thumbnail"},
		logger:   cfg.GetLogger().With("component", "view_handler"),
	}
}

func authAndRender(c fiber.Ctx, turnstileSiteKey string, pageContent templ.Component) error {
	claims := authHelper.GetClaims(c)
	headerButton := auth.AuthModal(turnstileSiteKey)
	if claims != nil {
		headerButton = menu.AvatarMenu(claims)
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return templates.BaseLayout(header.Header(headerButton), pageContent).Render(c.Context(), c.Response().BodyWriter())
}

func (h *handler) RenderPostPage(c fiber.Ctx) error {
	postIDStr := c.Params("id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		h.logger.Warn("invalid post ID format", "id", postIDStr, "error", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid post ID")
	}

	// Fetch main post
	post, err := h.svc.Get(c.Context(), postID)
	if err != nil {
		h.logger.Error("failed to retrieve post", "id", postID, "error", err)
		return fiber.NewError(fiber.StatusNotFound, "Post not found")
	}

	// Parse category ID safely
	categoryID, err := uuid.Parse(post.CategoryID)
	if err != nil {
		h.logger.Error("invalid category ID in post record", "id", postID, "category_id", post.CategoryID, "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Internal data corruption")
	}

	// Fetch related posts
	relatedPosts, err := h.svc.ListRelated(c.Context(), database.ListRelatedPostsParams{
		ID:         postID,
		Tags:       post.Tags,
		CategoryID: categoryID,
	})
	if err != nil {
		h.logger.Error("failed to fetch related posts", "id", postID, "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Could not load related content")
	}
	return authAndRender(c, h.cfg.Get().Cloudflare.TurnstileSiteKey, details.Details(post, relatedPosts, authHelper.GetClaims(c), h.variants))
}

func (h *handler) RenderIndexPage(c fiber.Ctx) error {
	// 1. Extract params
	query := c.Query("q", "")
	cols := strings.Split(c.Query("cols", "title"), ",")
	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	// 2. Prepare dynamic filter
	params := database.SearchPostsParams{
		SearchQuery:       query,
		SearchTitle:       slices.Contains(cols, "title"),
		SearchAuthor:      slices.Contains(cols, "author"),
		SearchDescription: slices.Contains(cols, "description"),
		SearchCategory:    slices.Contains(cols, "category"),
		MatchAll:          c.Query("match_all", "false") == "true",
		Tags:              strings.Split(c.Query("tags", ""), ","),
		Limit:             int32(limit),
		Offset:            int32(offset),
	}

	h.logger.Info("rendering index page", "query", query, "limit", limit, "offset", offset)

	// 3. Fetch posts using the updated service signature
	posts, total, err := h.svc.List(c.Context(), int32(limit), int32(offset))
	/*posts, total, err := h.svc.Search(c.Context(), params)*/

	if err != nil {
		h.logger.Error("list failed", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	return authAndRender(c, h.cfg.Get().Cloudflare.TurnstileSiteKey, home.HomePage(posts, total, &params))
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	app.Get("/", authHelper.HasSession(h.cfg), h.RenderIndexPage)
	app.Get("/post/:id/:slug/", authHelper.HasSession(h.cfg), h.RenderPostPage)
}
