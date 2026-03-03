package config

import (
	"comics-galore-web/internal/database"
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Service interface {
	Get() *Env
	Reload(ctx context.Context) error
	GetLogger() *slog.Logger
	GetQuerier() *database.Queries
	GetDbResource() database.Resources
}

type service struct {
	mu         sync.RWMutex
	cfg        *Env
	logger     *slog.Logger
	querier    *database.Queries
	dbResource database.Resources
}

func NewService(ctx context.Context) (Service, error) {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	s := &service{
		logger: logger,
	}

	if err := s.Reload(ctx); err != nil {
		return nil, fmt.Errorf("initial config load failed: %w", err)
	}

	dbResource, err := database.NewResources(ctx, s.cfg.DatabaseDSN, logger)
	if err != nil {
		return nil, fmt.Errorf("database initialization failure: %w", err)
	}

	s.dbResource = dbResource
	s.querier = database.New(dbResource.GetPool())

	return s, nil
}

func (s *service) Get() *Env {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *service) Reload(ctx context.Context) error {
	l := s.logger.With("op", "Reload")

	// 1. Load environment
	if err := godotenv.Overload(); err != nil {
		l.Debug("no .env file loaded", "error", err)
	}

	tempCfg := &Env{}
	if err := env.Parse(tempCfg); err != nil {
		return fmt.Errorf("parsing env: %w", err)
	}

	// 2. Properly context-aware JWKS initialization
	fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	jwks, err := keyfunc.NewDefaultCtx(fetchCtx, []string{tempCfg.JwksUrl})
	if err != nil {
		return fmt.Errorf("jwks initialization (network fetch failed): %w", err)
	}

	tempCfg.JwksFunc = jwks

	// 3. Atomic swap
	s.mu.Lock()
	s.cfg = tempCfg
	s.mu.Unlock()

	l.Info("configuration synchronized")
	return nil
}

func (s *service) GetLogger() *slog.Logger           { return s.logger }
func (s *service) GetQuerier() *database.Queries     { return s.querier }
func (s *service) GetDbResource() database.Resources { return s.dbResource }
