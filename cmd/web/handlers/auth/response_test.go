package auth

import (
	"comics-galore-web/internal/config"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHandleAuthResponse(t *testing.T) {
	app := fiber.New()

	mockCfg := config.NewMockService()

	// Initialize your handler with necessary mocks (Config, Logger, etc.)
	h := &handler{
		cfg: mockCfg,
	}

	// 3. Set the expectation ONCE with the backend.URL
	mockCfg.On("Get").Return(&config.Env{
		BetterAuth:       "better-auth",
		BetterAuthSecret: "your-mock-secret",
	})

	// Mock the logger if the handler or proxy uses it
	mockCfg.On("GetLogger").Return(slog.Default())

	t.Run("Should intercept 401 and render error fragment for HTMX", func(t *testing.T) {
		// Register a route that uses our middleware
		app.Post("/api/v1/auth/sign-in/email",
			// 1. Setup step: inject Locals that your middleware expects
			func(c fiber.Ctx) error {
				c.Locals("form", "sign-in")
				return c.Next()
			},
			h.handleAuthResponse,
			func(c fiber.Ctx) error {
				// Simulate the Proxy returning a 401 Unauthorized
				return c.Status(fiber.StatusUnauthorized).SendString(`{"message":"unauthorized"}`)
			})

		req := httptest.NewRequest("POST", "/api/v1/auth/sign-in/email", nil)
		req.Header.Set("HX-Request", "true")

		resp, _ := app.Test(req)

		// Assertions
		assert.Equal(t, fiber.StatusOK, resp.StatusCode) // Your code swaps 401 for 200 for HTMX
		assert.Equal(t, "authError", resp.Header.Get("HX-Trigger"))
		assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))
		assert.Equal(t, "#form-error", resp.Header.Get("HX-Retarget"))
		assert.Equal(t, "#form-error", resp.Header.Get("HX-Retarget"))
	})
}
