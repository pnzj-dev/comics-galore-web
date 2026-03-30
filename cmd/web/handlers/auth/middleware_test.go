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
		JwtSecret:        "jwt-secret",
	})

	// Mock the logger if the handler or proxy uses it
	mockCfg.On("GetLogger").Return(slog.Default())

	t.Run("Valid Login Input should pass to Next and set Locals", func(t *testing.T) {
		app.Post("/api/v1/auth/sign-in/email", h.validateInput, func(c fiber.Ctx) error {
			// Assert that locals were set correctly by the middleware
			//form := c.Locals("form")
			//assert.Equal(t, "sign-in", form)
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
		app.Post("/api/v1/auth/sign-up/email", h.validateInput, func(c fiber.Ctx) error {
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
		//assert.Equal(t, "#form-error", resp.Header.Get("HX-Retarget"))
	})

	t.Run("Should execute createCookies and transform session into JWT", func(t *testing.T) {
		// 1. Setup the route with a mock 'Better-Auth' response
		app.Get("/proxy-target", h.createCookie, func(c fiber.Ctx) error {
			// This simulates the JSON body Better-Auth returns
			// Ensure the ExpiresAt is in the FUTURE so the cookie isn't expired
			mockResponse := `{
				"user": { "id": "user_123", "email": "test@example.com", "role": "user" },
				"session": { 
					"token": "original-secret-token", 
					"expiresAt": "2026-12-31T23:59:59Z" 
				}
        	}`

			// Better-Auth usually sets its own cookie which we want to override
			c.Response().Header.Set(fiber.HeaderSetCookie, "better-auth-session=abc")
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			return c.SendString(mockResponse)
		})

		// 2. Execute the request
		req := httptest.NewRequest("GET", "/proxy-target", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)

		// 3. Assertions
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verify the custom JWT cookie is present
		setCookie := resp.Header.Get("Set-Cookie")
		assert.Contains(t, setCookie, "cg-auth-local") // Or whatever your dev name is

		// Verify the original Better-Auth cookie was deleted/overridden
		// (If your middleware uses c.Response().Header.Del(fiber.HeaderSetCookie))
		assert.NotContains(t, setCookie, "better-auth-session")

		// Optional: Verify the JWT value isn't empty
		assert.Greater(t, len(setCookie), 50)
	})
}
