package auth

import (
	"bytes"
	"comics-galore-web/internal/auth"
	"compress/gzip"
	"encoding/json"
	"github.com/valyala/fasthttp"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestSyncCookies(t *testing.T) {
	app := fiber.New()
	h := &handler{} // Minimal handler for testing

	t.Run("Successfully syncs cookie from valid JWT and Session data", func(t *testing.T) {
		c := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(c)

		// 1. Setup Mock Response Headers
		jwtValue := "mock-token-123"
		c.Response().Header.Set("Set-Auth-Jwt", jwtValue)

		// 2. Setup Mock Session Body
		expiry := time.Now().Add(24 * time.Hour)
		sessionData := auth.GetSession{
			Session: auth.Session{ExpiresAt: expiry},
		}
		body, _ := json.Marshal(sessionData)
		c.Response().SetBody(body)

		// 3. Execute
		err := h.syncCookies(c)
		assert.NoError(t, err)

		// 4. Assertions
		setCookieHeader := string(c.Response().Header.Peek("Set-Cookie"))

		assert.Contains(t, setCookieHeader, "comics-galore-jwt="+jwtValue)
		assert.Contains(t, setCookieHeader, "path=/")
		assert.Contains(t, setCookieHeader, "HttpOnly")
	})

	t.Run("Handles Gzip compressed response body", func(t *testing.T) {
		c := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(c)

		jwtValue := "gzip-token-456"
		c.Response().Header.Set("Set-Auth-Jwt", jwtValue)
		c.Response().Header.Set("Content-Encoding", "gzip")

		// Create Gzipped body
		sessionData := auth.GetSession{
			Session: auth.Session{ExpiresAt: time.Now().Add(time.Hour)},
		}
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_ = json.NewEncoder(gz).Encode(sessionData)
		err := gz.Close()
		if err != nil {
			return
		}
		c.Response().SetBody(buf.Bytes())

		err = h.syncCookies(c)
		assert.NoError(t, err)

		// Verify the response header contains the cookie
		setCookie := string(c.Response().Header.Peek("Set-Cookie"))
		assert.Contains(t, setCookie, "comics-galore-jwt="+jwtValue)
	})

	t.Run("Returns nil if Set-Auth-Jwt header is missing", func(t *testing.T) {
		c := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(c)

		err := h.syncCookies(c)

		assert.NoError(t, err)
		assert.Empty(t, c.Cookies("comics-galore-jwt"))
	})
}
