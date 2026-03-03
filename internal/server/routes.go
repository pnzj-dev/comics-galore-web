package server

import (
	"comics-galore-web/cmd/web"
	"comics-galore-web/internal/archive"
	"comics-galore-web/internal/auth"
	"comics-galore-web/internal/blog"
	"comics-galore-web/internal/comment"
	"comics-galore-web/internal/event"
	"comics-galore-web/internal/messaging"
	"comics-galore-web/internal/picture"
	"comics-galore-web/internal/qrcode"
	"comics-galore-web/internal/view"
	"comics-galore-web/internal/websocket"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"io/fs"
)

func (s *FiberServer) RegisterFiberRoutes(deps *Deps) {

	authHandler := auth.NewHandler(deps.Config)
	viewHandler := view.NewHandler(deps.Config, deps.Blog)
	blogHandler := blog.NewHandler(deps.Config, deps.Blog)
	eventHandler := event.NewHandler(deps.Config, deps.Broadcaster)
	qrcodeHandler := qrcode.NewHandler(deps.Config, deps.QrCode)
	commentHandler := comment.NewHandler(deps.Config, deps.Comment)
	pictureHandler := picture.NewHandler(deps.Config, deps.Picture)
	archiveHandler := archive.NewHandler(deps.Config, deps.Archive)
	messagingHandler := messaging.NewHandler(deps.Config, deps.Messaging)
	websocketHandler := websocket.NewHandler(deps.Config)

	//TODO: to delete or comment when debugging is finish
	s.App.Use(func(c fiber.Ctx) error {
		fmt.Printf("Method: %s | Path: %s\n", c.Method(), c.Path())
		return c.Next()
	})

	s.setupGlobalMiddleware(deps.Config)

	assetsFS, err := fs.Sub(web.Files, "assets")
	if err != nil {
		deps.Config.GetLogger().Error("failed to create assets sub-fs", "error", err)
	}

	s.App.Get("/assets/*", static.New("", static.Config{
		FS: assetsFS,
	}))

	//s.App.Get("/assets/*", static.New("", static.Config{FS: web.Files}))

	viewHandler.RegisterRoutes(s.App)
	authHandler.RegisterRoutes(s.App)
	blogHandler.RegisterRoutes(s.App)
	eventHandler.RegisterRoutes(s.App)
	qrcodeHandler.RegisterRoutes(s.App)
	archiveHandler.RegisterRoutes(s.App)
	pictureHandler.RegisterRoutes(s.App)
	commentHandler.RegisterRoutes(s.App)
	messagingHandler.RegisterRoutes(s.App)
	websocketHandler.RegisterRoutes(s.App)

}
