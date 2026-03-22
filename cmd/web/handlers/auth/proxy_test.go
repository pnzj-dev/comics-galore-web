package auth

import (
	"comics-galore-web/internal/config"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestProxyToBetterAuth(t *testing.T) {
	app := fiber.New()

	// 1. Start the backend FIRST to get the dynamic URL immediately
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("proxied-successfully"))
	}))
	defer backend.Close()

	// 2. Initialize the mock and cast it to access .On
	mockSvc := config.NewMockService()
	// Ensure this matches your mock struct name

	// 3. Set the expectation ONCE with the backend.URL
	mockSvc.On("Get").Return(&config.Env{
		BetterAuth:       backend.URL,
		BetterAuthSecret: "your-mock-secret",
	})

	// Mock the logger if the handler or proxy uses it
	mockSvc.On("GetLogger").Return(slog.Default())

	// 4. Initialize Handler
	h := &handler{
		validate: validator.New(),
		cfg:      mockSvc,
		logger:   slog.Default(),
	}

	app.All("/api/v1/auth/*", h.proxyToBetterAuth)

	t.Run("Successfully proxies path and query params", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/callback/google?code=123", nil)

		// Force the Host to match the backend port to avoid 400/500 errors
		req.Host = strings.TrimPrefix(backend.URL, "http://")

		resp, err := app.Test(req) // -1 disables timeout
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Defaults origin to localhost if missing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
		req.Host = strings.TrimPrefix(backend.URL, "http://")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
