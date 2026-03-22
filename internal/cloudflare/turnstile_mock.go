package cloudflare

import (
	"context"
	"github.com/stretchr/testify/mock"
)

type MockTurnstile struct {
	mock.Mock
}

// Ensure MockTurnstile implements the Turnstile interface at compile time
var _ Turnstile = (*MockTurnstile)(nil)

func NewMockTurnstile() *MockTurnstile {
	return new(MockTurnstile)
}

func (m *MockTurnstile) Verify(ctx context.Context, token, secretKey, remoteIP string) (*TurnstileResponse, error) {
	args := m.Called(ctx, token, secretKey, remoteIP)

	resp, _ := args.Get(0).(*TurnstileResponse)

	return resp, args.Error(1)
}
