package server

import (
	"comics-galore-web/internal/archive"
	"comics-galore-web/internal/blog"
	"comics-galore-web/internal/broadcaster"
	"comics-galore-web/internal/cloudflare"
	"comics-galore-web/internal/comment"
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/database"
	"comics-galore-web/internal/email"
	"comics-galore-web/internal/messaging"
	"comics-galore-web/internal/nowpayments"
	"comics-galore-web/internal/picture"
	"comics-galore-web/internal/qrcode"
	"github.com/gofiber/fiber/v3"
	"log/slog"
)

type Deps struct {
	Logger *slog.Logger
	DB     database.Resources

	Blog        blog.Service
	Email       email.Service
	QrCode      qrcode.Service
	Config      config.Service
	Archive     archive.Service
	Picture     picture.Service
	Comment     comment.Service
	Messaging   messaging.Service
	Cloudflare  cloudflare.Images
	Nowpayments nowpayments.Service
	Broadcaster broadcaster.Service
}

type FiberServer struct {
	*fiber.App
	logger *slog.Logger
	db     database.Resources

	blog        blog.Service
	email       email.Service
	qrcode      qrcode.Service
	config      config.Service
	archive     archive.Service
	picture     picture.Service
	comment     comment.Service
	messaging   messaging.Service
	cloudflare  cloudflare.Images
	nowpayments nowpayments.Service
	broadcaster broadcaster.Service
}

func (d *Deps) New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "comics-galore-web",
			AppName:      "comics-galore-web",
		}),
		logger:      d.Logger.With("component", "fiber_server"),
		db:          d.DB,
		blog:        d.Blog,
		email:       d.Email,
		config:      d.Config,
		qrcode:      d.QrCode,
		archive:     d.Archive,
		picture:     d.Picture,
		comment:     d.Comment,
		messaging:   d.Messaging,
		broadcaster: d.Broadcaster,
		nowpayments: d.Nowpayments,
		cloudflare:  d.Cloudflare,
	}

	return server
}
