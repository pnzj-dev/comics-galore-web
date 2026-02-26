package config

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Service interface {
	Get() *Env
	Reload(ctx context.Context) error
}

type service struct {
	mu     sync.RWMutex
	cfg    *Env
	logger *slog.Logger
}

func NewService(logger *slog.Logger) (Service, error) {
	s := &service{
		logger: logger.With("component", "config_service"),
	}

	// Initial load
	if err := s.Reload(context.Background()); err != nil {
		return nil, fmt.Errorf("initial config load failed: %w", err)
	}

	return s, nil
}

// Get returns a thread-safe copy of the configuration
func (s *service) Get() *Env {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Reload re-reads environment variables and re-initializes dependencies like JWKS
func (s *service) Reload(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	l := s.logger.With("op", "Reload")

	// 1. Optional: Re-load .env (useful for local dev)
	if err := godotenv.Overload(); err != nil {
		l.Debug("skipping .env overload", "reason", "file not found or unreadable")
	}

	newCfg := &Env{}
	if err := env.Parse(newCfg); err != nil {
		l.Error("environment parsing failed", "error", err)
		return err
	}

	// 2. Logic-heavy initializations (like JWKS)
	jwksFunc, err := keyfunc.NewDefault([]string{newCfg.JwksUrl})
	if err != nil {
		l.Error("failed to update JWKS keyfunc", "url", newCfg.JwksUrl, "error", err)
		return err
	}
	newCfg.JwksFunc = jwksFunc

	s.cfg = newCfg
	l.Info("configuration synchronized successfully")
	return nil
}
