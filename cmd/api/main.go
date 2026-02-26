package main

import (
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/database"
	"comics-galore-web/internal/messaging"
	"comics-galore-web/internal/picture"
	"comics-galore-web/internal/server"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 1. Initialize Structured Logger (JSON for production)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger) // Set as default for any global calls

	// 2. Setup root context for signal trapping
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Config Service
	cfg, err := config.NewService(logger)
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 4. Initialize Database Resources
	dbResource, err := database.NewResources(ctx, cfg.Get().DatabaseDSN)
	if err != nil {
		logger.Error("critical database initialization failure", "error", err)
		os.Exit(1)
	}

	// 5. Initialize Domain Services (Injecting logger and config)
	picSvc := picture.NewService(cfg, logger, false)

	// Assuming your server struct or a separate messaging service handles WebSockets
	// msgSvc := messaging.NewService(dbResource.Pool, logger)

	pool := dbResource.GetPool()

	deps := server.Deps{
		Config:    cfg,
		Logger:    logger,
		DB:        dbResource,
		Picture:   picture.NewService(cfg, logger, false),
		Messaging: messaging.NewService(pool, logger),
	}

	// 6. Provision Server
	internalServer := deps.New()
	internalServer.RegisterFiberRoutes()

	// 7. Start Server
	serverErr := make(chan error, 1)
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}

		logger.Info("starting server", "port", port, "env", os.Getenv("APP_ENV"))
		serverErr <- internalServer.Listen(":" + port)
	}()

	// 8. Wait for Signal or Error
	select {
	case err := <-serverErr:
		if err != nil {
			logger.Error("server startup failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	// 9. GRACEFUL SHUTDOWN SEQUENCE (Sequential)
	// We use a dedicated timeout context for the entire shutdown phase
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// A. Shutdown HTTP Server (Stop accepting new requests/WebSocket upgrades)
	logger.Info("shutting down HTTP server...")
	if err := internalServer.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown error", "error", err)
	}

	// B. Shutdown Background Workers (Picture processing/S3 uploads)
	logger.Info("waiting for background workers to drain...")
	if err := picSvc.Shutdown(5 * time.Second); err != nil {
		logger.Warn("picture service workers failed to drain", "error", err)
	}

	// C. Shutdown WebSocket Hub (Disconnect clients gracefully)
	// If you have a specific Hub.Shutdown(), call it here.
	// internalServer.WS.Shutdown()

	// D. Close Database Pool (Last step: Ensure no queries are in-flight)
	logger.Info("closing database connections...")
	dbResource.Close(shutdownCtx)

	logger.Info("graceful shutdown complete. system exited.")
}
