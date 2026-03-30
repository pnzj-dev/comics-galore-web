package auth

import (
	"bytes"
	"comics-galore-web/internal/config"
	"compress/gzip"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http/httptest"
	"testing"
)

func TestCreateCookieMiddleware(t *testing.T) {
	// 1. Setup Mock Config
	mSvc := config.NewMockService()
	mSvc.On("Get").Return(&config.Env{
		AppEnv:    "development",
		JwtSecret: "test-32-character-secret-key-12345",
	})

	h := &handler{cfg: mSvc}
	app := fiber.New()

	t.Run("Success - Standard JSON", func(t *testing.T) {
		// Setup a route that mimics the proxy response
		app.Get("/test-success", h.createCookie, func(c fiber.Ctx) error {
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			return c.SendString(`{
                "user": {"id": "123", "email": "test@test.com", "role": "user"},
                "session": {"expiresAt": "2026-12-31T23:59:59Z", "token": "abc"}
            }`)
		})

		req := httptest.NewRequest("GET", "/test-success", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		// Verify cookie exists and is named correctly for dev
		assert.Contains(t, resp.Header.Get("Set-Cookie"), "cg-auth-local")
		// Verify body was cleared by ResetBody()
		body, _ := io.ReadAll(resp.Body)
		assert.Empty(t, body)
	})

	t.Run("Success - Gzipped JSON", func(t *testing.T) {
		app.Get("/test-gzip", h.createCookie, func(c fiber.Ctx) error {
			json := `{"user": {"id": "456"}, "session": {"expiresAt": "2026-12-31T23:59:59Z"}}`

			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			_, _ = gz.Write([]byte(json))
			_ = gz.Close()

			c.Set(fiber.HeaderContentEncoding, "gzip")
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			return c.Send(buf.Bytes())
		})

		req := httptest.NewRequest("GET", "/test-gzip", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Set-Cookie"), "cg-auth-local")
	})

	t.Run("Failure - Empty Body", func(t *testing.T) {
		app.Get("/test-empty", h.createCookie, func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest("GET", "/test-empty", nil)
		resp, _ := app.Test(req)

		// Middleware should return nil and not set a cookie if body is empty
		assert.Empty(t, resp.Header.Get("Set-Cookie"))
	})

	t.Run("Failure - Invalid JSON", func(t *testing.T) {
		app.Get("/test-invalid", h.createCookie, func(c fiber.Ctx) error {
			return c.SendString(`{invalid-json`)
		})

		req := httptest.NewRequest("GET", "/test-invalid", nil)
		resp, _ := app.Test(req)

		assert.Empty(t, resp.Header.Get("Set-Cookie"))
	})
}
