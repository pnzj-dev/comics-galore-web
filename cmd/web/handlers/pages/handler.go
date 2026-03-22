package pages

import (
	"comics-galore-web/cmd/web/components/header"
	"comics-galore-web/cmd/web/components/pagination"
	"comics-galore-web/cmd/web/views/pages"
	"comics-galore-web/internal/auth"
	"comics-galore-web/internal/blog"
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/database"
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type handler struct {
	svc        blog.Service
	cfg        config.Service
	logger     *slog.Logger
	categories []database.Category
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
	RenderPostPage(c fiber.Ctx) error
	RenderIndexPage(c fiber.Ctx) error
	RenderSearchPage(c fiber.Ctx) error
	RenderAuthorsPage(c fiber.Ctx) error
	RenderTopRatedPage(c fiber.Ctx) error
	RenderCategoryPage(c fiber.Ctx) error
}

func (h *handler) RenderAuthorsPage(c fiber.Ctx) error {
	h.logger.Warn("hit unimplemented route", "path", c.Path(), "handler", "RenderAuthorsPage")
	return h.renderNotImplemented(c, "Authors Page")
}

func (h *handler) RenderTopRatedPage(c fiber.Ctx) error {
	h.logger.Warn("hit unimplemented route", "path", c.Path(), "handler", "RenderTopRatedPage")
	return h.renderNotImplemented(c, "Top Rated Page")
}

// Helper to provide a consistent "Work in Progress" UI
func (h *handler) renderNotImplemented(c fiber.Ctx, featureName string) error {
	component := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, fmt.Sprintf(`
            <div class="flex flex-col items-center justify-center p-20 border-4 border-dashed border-gray-200 rounded-2xl m-8">
                <h1 class="text-2xl font-bold text-gray-400 uppercase tracking-widest">Coming Soon</h1>
                <p class="text-gray-500 mt-2">The <strong>%s</strong> is currently under construction.</p>
                <a href="/" class="mt-6 text-blue-600 hover:underline">← Back to Home</a>
            </div>
        `, featureName))
		return err
	})

	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return component.Render(c.Context(), c.Response().BodyWriter())
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	viewGroup := app.Group("/", auth.HasSession(h.cfg), h.ViewMiddleware)
	viewGroup.Get("/", h.RenderIndexPage)
	viewGroup.Get("/search", h.RenderSearchPage)
	viewGroup.Get("/authors", h.RenderAuthorsPage)
	viewGroup.Get("/top-rated", h.RenderTopRatedPage)
	viewGroup.Get("/post/:id/:slug/", h.RenderPostPage)
	viewGroup.Get("/category/:category/:uuid?", h.RenderCategoryPage)
}

func NewHandler(cfg config.Service, svc blog.Service) Handler {
	// Use Background only for init; real requests should use c.Context()
	categories, err := svc.ListCategories(context.Background())
	logger := cfg.GetLogger().With("component", "view_handler")

	if err != nil {
		logger.Error("critical: failed to preload categories", "error", err)
	}

	return &handler{
		svc:        svc,
		cfg:        cfg,
		logger:     logger,
		categories: categories,
	}
}

// Helper to reduce boilerplate in every render function
func (h *handler) render(c fiber.Ctx, component templ.Component) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return component.Render(c.Context(), c.Response().BodyWriter())
}

func (h *handler) RenderPostPage(c fiber.Ctx) error {
	idParam := c.Params("id")
	postID, err := uuid.Parse(idParam)
	if err != nil {
		h.logger.Warn("invalid post uuid", "id", idParam)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid post ID")
	}

	post, err := h.svc.Get(c.Context(), postID)
	if err != nil {
		h.logger.Error("post retrieval failed", "id", postID, "error", err)
		return fiber.NewError(fiber.StatusNotFound, "Post not found")
	}

	// Related posts logic
	categoryID, _ := uuid.Parse(post.CategoryID)
	relatedPosts, _ := h.svc.ListRelated(c.Context(), database.ListRelatedPostsParams{
		ID:         postID,
		Tags:       post.Tags,
		CategoryID: categoryID,
	})

	return h.render(c, pages.Details(post, relatedPosts, h.createNavItems(h.categories, c.Path())))
}

func (h *handler) RenderSearchPage(c fiber.Ctx) error {
	l, o := h.getPaginationParams(c)
	query := c.Query("q", "")
	cols := strings.Split(c.Query("cols", "title"), ",")

	var tags []string
	if t := c.Query("tags", ""); t != "" {
		tags = strings.Split(t, ",")
	}

	params := database.SearchPostsParams{
		SearchQuery:       query,
		SearchTitle:       slices.Contains(cols, "title"),
		SearchAuthor:      slices.Contains(cols, "author"),
		SearchDescription: slices.Contains(cols, "description"),
		SearchCategory:    slices.Contains(cols, "category"),
		MatchAll:          c.Query("match_all") == "true",
		Tags:              tags,
		Limit:             int32(l),
		Offset:            int32(o),
	}

	posts, total, err := h.svc.Search(c.Context(), params)
	if err != nil {
		h.logger.Error("search operation failed", "params", params, "error", err)
	}

	props := h.createPaginationProps(l, o, total, c.Path())
	return h.render(c, pages.Home(posts, h.createNavItems(h.categories, c.Path()), props))
}

func (h *handler) RenderIndexPage(c fiber.Ctx) error {
	l, o := h.getPaginationParams(c)

	posts, total, err := h.svc.List(c.Context(), int32(l), int32(o))
	if err != nil {
		h.logger.Error("index list failed", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	props := h.createPaginationProps(l, o, total, c.Path())

	h.logger.Debug("index page rendered", "count", len(posts), "total", total)
	return h.render(c, pages.Home(posts, h.createNavItems(h.categories, c.Path()), props))
}

func (h *handler) RenderCategoryPage(c fiber.Ctx) error {
	uuidParam := c.Params("uuid")
	if uuidParam == "" {
		return c.Redirect().To("/")
	}

	l, o := h.getPaginationParams(c)
	posts, total, err := h.svc.ListByCategory(c.Context(), int32(l), int32(o), uuidParam)
	if err != nil {
		h.logger.Error("category list failed", "uuid", uuidParam, "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	props := h.createPaginationProps(l, o, total, c.Path())
	return h.render(c, pages.Home(posts, h.createNavItems(h.categories, c.Path()), props))
}

// Internal Helper for cleaner pagination
func (h *handler) getPaginationParams(c fiber.Ctx) (limit, offset int) {
	limit, _ = strconv.Atoi(c.Query("limit", "30"))
	offset, _ = strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 {
		limit = 30
	}
	return
}

func (h *handler) createPaginationProps(limit, offset int, totalRecords int64, path string) pagination.Props {

	currentPage := (offset / limit) + 1
	totalPages := int((totalRecords + int64(limit) - 1) / int64(limit))

	return pagination.Props{
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		HxTarget:    "#main-content",
		HxEndpoint:  path,
		HxPushURL:   true,
	}
}

func (h *handler) createNavItems(categories []database.Category, currentPath string) []header.NavItem {
	// Optimization: Pre-allocate capacity
	// Home (1) + Categories (N) + Top/Authors (2)
	navItems := make([]header.NavItem, 0, 1+len(categories)+2)

	navItems = append(navItems, header.NavItem{Label: "Home", Href: "/"})

	for _, cat := range categories {
		navItems = append(navItems, header.NavItem{
			Label: cat.DisplayName,
			Href:  "/category/" + cat.Slug,
		})
	}

	navItems = append(navItems,
		header.NavItem{Label: "Top-Rated", Href: "/top-rated"},
		header.NavItem{Label: "Authors", Href: "/authors"},
	)

	// Active state logic
	for i := range navItems {
		if navItems[i].Href == "/" {
			navItems[i].Active = currentPath == "/"
		} else {
			navItems[i].Active = strings.HasPrefix(currentPath, navItems[i].Href)
		}
	}

	return navItems
}
