package main

import (
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
	"comics-galore-web/internal/server"
	"comics-galore-web/internal/storage"
	"context"
	"github.com/gofiber/fiber/v3/log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Config Service
	cfg, err := config.NewService(ctx)
	if err != nil {
		// Use standard log here because cfg is nil!
		log.Errorf("critical failure: failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Correct way to inspect your config service if you want to see values
	//cfg.GetLogger().Info("config service initialized", "env", cfg.Get())

	localBroadcaster := broadcaster.NewService(cfg)

	if cfg.GetLogger() == nil {
		log.Errorf("critical failure: logger is not configured")
		os.Exit(1)
	}

	deps := server.Deps{
		Config:      cfg,
		Blog:        blog.NewService(cfg.GetQuerier(), cfg.GetLogger()),
		Email:       email.NewService(cfg),
		QrCode:      qrcode.NewService(cfg),
		Storage:     storage.NewService(cfg),
		Picture:     picture.NewService(cfg, false),
		Comment:     comment.NewService(cfg, localBroadcaster),
		Messaging:   messaging.NewService(cfg),
		Cloudflare:  cloudflare.NewService(cfg),
		Nowpayments: nowpayments.NewService(cfg),
		Turnstile:   cloudflare.NewTurnstile(cfg),
		Broadcaster: localBroadcaster,
	}

	// 6. Provision Server
	internalServer := server.New(&deps)
	internalServer.RegisterFiberRoutes(&deps)

	// 7. Start Server
	serverErr := make(chan error, 1)
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}

		cfg.GetLogger().Info("starting server", "port", port, "env", os.Getenv("APP_ENV"))
		serverErr <- internalServer.Listen(":" + port)
	}()

	// 8. Wait for Signal or Error
	select {
	case err := <-serverErr:
		if err != nil {
			cfg.GetLogger().Error("server startup failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		cfg.GetLogger().Info("shutdown signal received")
	}

	// 9. GRACEFUL SHUTDOWN SEQUENCE (Sequential)
	// We use a dedicated timeout context for the entire shutdown phase
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// A. Shutdown HTTP Server (Stop accepting new requests/WebSocket upgrades)
	cfg.GetLogger().Info("shutting down HTTP server...")
	if err := internalServer.ShutdownWithContext(shutdownCtx); err != nil {
		cfg.GetLogger().Error("HTTP shutdown error", "error", err)
	}

	// B. Shutdown Background Workers (Picture processing/S3 uploads)
	cfg.GetLogger().Info("waiting for background workers to drain...")
	if err := deps.Picture.Shutdown(5 * time.Second); err != nil {
		cfg.GetLogger().Warn("picture service workers failed to drain", "error", err)
	}

	// C. Shutdown WebSocket Hub (Disconnect clients gracefully)
	// If you have a specific Hub.Shutdown(), call it here.
	// internalServer.WS.Shutdown()

	// D. Close Database Pool (Last step: Ensure no queries are in-flight)
	cfg.GetLogger().Info("closing database connections...")
	cfg.GetDbResource().Close(shutdownCtx)

	cfg.GetLogger().Info("graceful shutdown complete. system exited.")
}
