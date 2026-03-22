package server

import (
	"comics-galore-web/cmd/web"
	"comics-galore-web/cmd/web/handlers/auth"
	"comics-galore-web/cmd/web/handlers/modal"
	"comics-galore-web/cmd/web/handlers/pages"
	"comics-galore-web/internal/blog"
	"comics-galore-web/internal/comment"
	"comics-galore-web/internal/event"
	"comics-galore-web/internal/messaging"
	"comics-galore-web/internal/picture"
	"comics-galore-web/internal/qrcode"
	"comics-galore-web/internal/storage"
	"comics-galore-web/internal/websocket"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	slogfiber "github.com/samber/slog-fiber"
	"io/fs"
)

func (s *FiberServer) RegisterFiberRoutes(deps *Deps) {

	modalHandler := modal.NewHandler(deps.Config)
	pagesHandler := pages.NewHandler(deps.Config, deps.Blog)
	blogHandler := blog.NewHandler(deps.Config, deps.Blog)
	authHandler := auth.NewHandler(deps.Config, deps.Turnstile)
	qrcodeHandler := qrcode.NewHandler(deps.Config, deps.QrCode)
	eventHandler := event.NewHandler(deps.Config, deps.Broadcaster)
	storageHandler := storage.NewHandler(deps.Config, deps.Storage)
	commentHandler := comment.NewHandler(deps.Config, deps.Comment)
	pictureHandler := picture.NewHandler(deps.Config, deps.Picture)
	messagingHandler := messaging.NewHandler(deps.Config, deps.Messaging)
	websocketHandler := websocket.NewHandler(deps.Config)

	/*s.App.Use(func(c fiber.Ctx) error {
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://challenges.cloudflare.com; frame-src 'self' https://challenges.cloudflare.com; connect-src 'self' https://challenges.cloudflare.com;")
		return c.Next()
	})*/
	s.App.Use(slogfiber.New(deps.Config.GetLogger()))

	/*s.App.Use(slogfiber.NewWithConfig(deps.Config.GetLogger(), slogfiber.Config{
	DefaultLevel:     slog.LevelInfo,
	ClientErrorLevel: slog.LevelWarn,  // 4xx
	ServerErrorLevel: slog.LevelError, // 5xx

	WithUserAgent: true, // Logs User-Agent by default
	WithRequestID: true, // Assumes you have a request ID middleware
	WithClientIP:  true, // Logs c.IP()

	// Optional: enable request/response body/header logging (careful with sensitive data!)
	// WithRequestBody:    true,
	// WithResponseBody:   true,

	// Filters (skip logging certain requests)
	Filters: []slogfiber.Filter{
		// Built-in helpers
		slogfiber.IgnoreStatus(404),     // Skip 404s
		slogfiber.IgnorePath("/health"), // Skip health checks
		slogfiber.IgnorePath("/metrics"),

		// Custom filter func (equivalent to what I suggested before)
		func(c fiber.Ctx) bool {
			return c.Path() == "/static/*" // Skip static files
		},
	}}))*/

	//TODO: to delete or comment when debugging is finish
	s.App.Use(func(c fiber.Ctx) error {
		fmt.Printf("Method: %s | Path: %s\n", c.Method(), c.Path())
		return c.Next()
	})

	s.setupGlobalMiddleware(deps.Config)
	s.setupRateLimiting(deps.Config)

	assetsFS, err := fs.Sub(web.Files, "assets")
	if err != nil {
		deps.Config.GetLogger().Error("failed to create assets sub-fs", "error", err)
	}

	s.App.Get("/assets/*", static.New("", static.Config{FS: assetsFS}))

	s.App.Get("/health", func(c fiber.Ctx) error { return c.SendString("OK") })

	//s.App.Get("/assets/*", static.New("", static.Config{FS: web.Files}))

	pagesHandler.RegisterRoutes(s.App)
	authHandler.RegisterRoutes(s.App)
	modalHandler.RegisterRoutes(s.App)
	blogHandler.RegisterRoutes(s.App)
	eventHandler.RegisterRoutes(s.App)
	qrcodeHandler.RegisterRoutes(s.App)
	storageHandler.RegisterRoutes(s.App)
	pictureHandler.RegisterRoutes(s.App)
	commentHandler.RegisterRoutes(s.App)
	messagingHandler.RegisterRoutes(s.App)
	websocketHandler.RegisterRoutes(s.App)

}
