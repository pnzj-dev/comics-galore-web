package config

import (
	"comics-galore-web/internal/database"
	"context"
	"github.com/stretchr/testify/mock"
	"log/slog"
)

type MockService struct {
	mock.Mock
}

// NewMockService returns the interface type Service
func NewMockService() *MockService {
	return new(MockService)
}

func (m *MockService) Get() *Env {
	args := m.Called()
	return args.Get(0).(*Env)
}

func (m *MockService) Reload(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockService) GetLogger() *slog.Logger {
	args := m.Called()
	return args.Get(0).(*slog.Logger)
}

func (m *MockService) GetQuerier() *database.Queries {
	args := m.Called()
	return args.Get(0).(*database.Queries)
}

func (m *MockService) GetDbResource() database.Resources {
	args := m.Called()
	return args.Get(0).(database.Resources)
}
