package auth

import (
	"bytes"
	"comics-galore-web/internal/config"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestValidateAuthInput(t *testing.T) {
	app := fiber.New()

	mockCfg := config.NewMockService()
	h := &handler{
		validate: validator.New(),
		cfg:      mockCfg,
		logger:   slog.Default(),
	}
	// 3. Set the expectation ONCE with the backend.URL
	mockCfg.On("Get").Return(&config.Env{
		BetterAuth:       "better-auth-url",
		BetterAuthSecret: "your-mock-secret",
	})

	// Mock the logger if the handler or proxy uses it
	mockCfg.On("GetLogger").Return(slog.Default())

	t.Run("Valid Login Input should pass to Next and set Locals", func(t *testing.T) {
		app.Post("/api/v1/auth/sign-in/email", h.validateInput(), func(c fiber.Ctx) error {
			// Assert that locals were set correctly by the middleware
			form := c.Locals("form")
			assert.Equal(t, "sign-in", form)
			return c.SendStatus(fiber.StatusOK)
		})

		payload := map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/auth/sign-in/email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Invalid Email should trigger validation error", func(t *testing.T) {
		// We mock the route. If it reaches the final handler, the test fails.
		app.Post("/api/v1/auth/sign-up/email", h.validateInput(), func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusExpectationFailed)
		})

		payload := map[string]string{
			"email":    "not-an-email",
			"password": "short",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/auth/sign-up/email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HX-Request", "true") // To trigger HTMX logic in renderError

		resp, _ := app.Test(req)

		// Your renderError sets status 200 for HTMX swaps
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		assert.Equal(t, "#form-error", resp.Header.Get("HX-Retarget"))
	})
}

func TestWithCookieSync(t *testing.T) {
	app := fiber.New()

	// We need a way to mock syncCookies. Since it's a method, we
	// expect it to be called after the final handler.
	h := &handler{}

	t.Run("Should execute syncCookies after successful next handler", func(t *testing.T) {
		app.Get("/proxy-target", h.withCookieSync(), func(c fiber.Ctx) error {
			c.Response().Header.Set("Set-Auth-Jwt", "test-token")
			return c.SendString("proxy-response")
		})

		req := httptest.NewRequest("GET", "/proxy-target", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		// Verify that syncCookies actually ran by checking for the cookie
		// Note: Use the logic we discussed in previous steps to check Response Headers
		setCookie := resp.Header.Get("Set-Cookie")
		assert.Contains(t, setCookie, "comics-galore-jwt")
	})
}
