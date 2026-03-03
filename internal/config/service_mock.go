package config

import (
	"comics-galore-web/internal/database"
	"context"
	"log/slog"
	"os"
)

// MockService satisfies the config.Service interface for testing purposes.
type MockService struct {
	// CustomEnv allows you to define exactly what Get() returns per test.
	CustomEnv *Env
	// ReloadErr allows you to simulate a failure during a config reload.
	ReloadErr error
	// TrackReloads counts how many times Reload was called.
	ReloadCalls int

	// Fields for the interface methods
	Logger    *slog.Logger
	Querier   *database.Queries
	Resources database.Resources
}

// NewMockService creates a mock with sane defaults for basic testing.
func NewMockService(initialEnv *Env) *MockService {
	if initialEnv == nil {
		initialEnv = &Env{}
	}
	return &MockService{
		CustomEnv: initialEnv,
		// Default logger to avoid nil pointer panics during tests
		Logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

// Get returns the stubbed configuration.
func (m *MockService) Get() *Env {
	return m.CustomEnv
}

// GetLogger returns the stubbed logger.
func (m *MockService) GetLogger() *slog.Logger {
	return m.Logger
}

// GetQuerier returns the stubbed database queries.
func (m *MockService) GetQuerier() *database.Queries {
	return m.Querier
}

// GetDbResource returns the stubbed database resources (pools, connections).
func (m *MockService) GetDbResource() database.Resources {
	return m.Resources
}

// Reload increments a counter and returns a stubbed error (if any).
func (m *MockService) Reload(ctx context.Context) error {
	m.ReloadCalls++
	return m.ReloadErr
}
