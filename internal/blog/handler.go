package blog

import (
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/database"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/google/uuid"
)

type handler struct {
	svc             Service
	logger          *slog.Logger
	viewRateLimiter fiber.Handler
}

type Handler interface {
	ListPosts(c fiber.Ctx) error
	SavePost(c fiber.Ctx) error
	GetPostDetail(c fiber.Ctx) error
	RegisterRoutes(app *fiber.App)
}

func NewHandler(cfg config.Service, svc Service) Handler {
	// 1. Initialize rate limiter for views
	viewLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP() + "_" + c.Params("id")
		},
		LimitReached: func(c fiber.Ctx) error {
			// Mark as limited but don't block the request
			c.Locals("is_rate_limited", true)
			return c.Next()
		},
	})

	return &handler{
		svc:             svc,
		logger:          cfg.GetLogger().With("component", "blog_handler"),
		viewRateLimiter: viewLimiter,
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	group := app.Group("/api/v1/post")

	group.Get("/list", h.ListPosts)
	group.Post("/", h.SavePost) // Added POST route which was missing registration

	// Middleware chain: Limit -> Track -> Handler
	group.Get("/:id", h.viewRateLimiter, h.GetPostDetail)
}

func (h *handler) GetPostDetail(c fiber.Ctx) error {
	idStr := c.Params("id")
	l := h.logger.With("op", "GetPostDetail", "post_id", idStr)

	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id format"})
	}

	post, err := h.svc.Get(c.Context(), id)
	if err != nil {
		l.Error("post lookup failed", "error", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	return c.JSON(post)
}

func (h *handler) ListPosts(c fiber.Ctx) error {

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	l := h.logger.With("op", "ListPosts", "limit", limit, "offset", offset)

	posts, total, err := h.svc.List(c.Context(), int32(limit), int32(offset))
	if err != nil {
		l.Error("failed to list posts", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to retrieve posts"})
	}

	return c.JSON(fiber.Map{
		"posts": posts,
		"total": total,
	})
}

func (h *handler) SavePost(c fiber.Ctx) error {
	l := h.logger.With("op", "SavePost")
	payload := new(database.UpsertPostParams)

	if err := c.Bind().Body(payload); err != nil {
		l.Warn("invalid request payload", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	savePost, err := h.svc.Save(c.Context(), *payload)
	if err != nil {
		l.Error("persistence failure", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save post"})
	}

	l.Info("post saved successfully", "post_id", savePost.ID)
	return c.Status(fiber.StatusCreated).JSON(savePost)
}
