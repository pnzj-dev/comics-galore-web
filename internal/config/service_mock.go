package config

import (
	"context"
)

// MockService satisfies the config.Service interface for testing purposes.
type MockService struct {
	// CustomEnv allows you to define exactly what Get() returns per test.
	CustomEnv *Env
	// ReloadErr allows you to simulate a failure during a config reload.
	ReloadErr error
	// TrackReloads counts how many times Reload was called.
	ReloadCalls int
}

// NewMockService creates a mock with sane defaults for basic testing.
func NewMockService(initialEnv *Env) *MockService {
	if initialEnv == nil {
		initialEnv = &Env{}
	}
	return &MockService{
		CustomEnv: initialEnv,
	}
}

// Get returns the stubbed configuration.
func (m *MockService) Get() *Env {
	return m.CustomEnv
}

// Reload increments a counter and returns a stubbed error (if any).
func (m *MockService) Reload(ctx context.Context) error {
	m.ReloadCalls++
	return m.ReloadErr
}
