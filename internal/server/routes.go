package server

import (
	"comics-galore-web/cmd/web"
	"comics-galore-web/internal/archive"
	"comics-galore-web/internal/auth"
	"comics-galore-web/internal/blog"
	"comics-galore-web/internal/comment"
	"comics-galore-web/internal/messaging"
	"comics-galore-web/internal/picture"
	"comics-galore-web/internal/qrcode"
	"comics-galore-web/internal/websocket"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func (s *FiberServer) RegisterFiberRoutes() {

	blogHandler := blog.NewHandler(s.blog, s.logger)
	authHandler := auth.NewHandler(s.config, s.logger)
	qrcodeHandler := qrcode.NewHandler(s.qrcode, s.logger)
	commentHandler := comment.NewHandler(s.comment, s.logger)
	pictureHandler := picture.NewHandler(s.picture, s.logger)
	archiveHandler := archive.NewHandler(s.archive, s.logger)
	messagingHandler := messaging.NewHandler(s.messaging, s.logger)
	websocketHandler := websocket.NewHandler(s.logger)

	s.App.Get("/assets/*", static.New("", static.Config{FS: web.Files}))

	s.setupGlobalMiddleware(s.config)
	authHandler.RegisterRoutes(s.App)
	blogHandler.RegisterRoutes(s.App)
	qrcodeHandler.RegisterRoutes(s.App)
	archiveHandler.RegisterRoutes(s.App)
	pictureHandler.RegisterRoutes(s.App)
	commentHandler.RegisterRoutes(s.App)
	messagingHandler.RegisterRoutes(s.App)
	websocketHandler.RegisterRoutes(s.App)

}
