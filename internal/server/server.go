package server

import (
	"comics-galore-web/internal/archive"
	"comics-galore-web/internal/blog"
	"comics-galore-web/internal/broadcaster"
	"comics-galore-web/internal/cloudflare"
	"comics-galore-web/internal/comment"
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/email"
	"comics-galore-web/internal/messaging"
	"comics-galore-web/internal/nowpayments"
	"comics-galore-web/internal/picture"
	"comics-galore-web/internal/qrcode"
	"fmt"
	"github.com/gofiber/fiber/v3"
)

type Deps struct {
	Blog        blog.Service
	Email       email.Service
	Config      config.Service
	QrCode      qrcode.Service
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
}

func New(d *Deps) *FiberServer {
	// 1. Get config once to avoid repeated map/struct lookups
	cfg := d.Config.Get()

	// 2. Prepare metadata strings
	appName := fmt.Sprintf("comics-galore-web (%s)", cfg.AppEnv)
	serverHeader := fmt.Sprintf("comics-galore-web/%s", cfg.Version)

	// 3. Initialize Fiber with optimized defaults
	app := fiber.New(fiber.Config{
		AppName:      appName,
		ServerHeader: serverHeader,
		// Optional: Good practice to include strict routing or custom error handlers here
		StrictRouting: true,
	})

	return &FiberServer{
		App: app,
	}
}
