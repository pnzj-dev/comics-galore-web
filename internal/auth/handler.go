package auth

import (
	"bytes"
	"comics-galore-web/internal/config"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3/log"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
)

type handler struct {
	cfg    config.Service
	logger *slog.Logger
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
	GetUserFromSession(c fiber.Ctx) error
}

func NewHandler(cfg config.Service) Handler {
	return &handler{
		cfg:    cfg,
		logger: cfg.GetLogger().With("component", "auth_proxy"),
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	// 1. Ensure targetBase is clean (e.g., "http://localhost:3000/api/auth")
	targetBase := strings.TrimSuffix(h.cfg.Get().BetterAuth, "/")

	h.logger.Info("registering unified auth proxy", "target", targetBase)

	group := app.Group("/api/v1/auth")

	group.All("/*", func(c fiber.Ctx) error {
		// Capture everything after /api/v1/auth/
		// If request is /api/v1/auth/sign-in/email, path is "sign-in/email"
		path := strings.TrimPrefix(c.Params("*"), "/")

		// 2. Construct final destination URL
		fullTarget := fmt.Sprintf("%s/%s", targetBase, path)

		// 3. Preserve Query Parameters (Essential for OAuth callbacks and email verification)
		if qs := string(c.Request().URI().QueryString()); qs != "" {
			fullTarget = fmt.Sprintf("%s?%s", fullTarget, qs)
		}

		// 4. Handle the Authorization: Bearer Header
		c.Request().Header.Set(fiber.HeaderAuthorization, "Bearer "+h.cfg.Get().BetterAuthSecret)

		h.logger.Debug("auth_proxy_request",
			"method", c.Method(),
			"original_path", c.Path(),
			"proxy_target", fullTarget,
		)

		origin := c.Get("Origin")

		// 2. If browser didn't send it (rare), use your Base URL
		if origin == "" {
			origin = "http://localhost:8080" // Your Go backend URL
		}
		c.Request().Header.Set("Origin", origin)

		// 5. Execute the proxy
		if err := proxy.Do(c, fullTarget); err != nil {
			h.logger.Error("auth_proxy_failed",
				"error", err,
				"target", fullTarget,
			)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"error": "authentication service unreachable",
			})
		}

		// --- DEBUG START ---
		log.Debug("--- PROXY RESPONSE DEBUG ---")

		// 1. Debug Headers
		setAuthJwt := ""
		for key, values := range c.Response().Header.All() {
			log.Debugf("%s: %s", string(key), string(values))
			if string(key) == "Set-Auth-Jwt" {
				setAuthJwt = string(values)
			}
		}
		bodyBytes := c.Response().Body()
		contentEncoding := string(c.Response().Header.Peek("Content-Encoding"))

		var finalBody []byte

		if contentEncoding == "gzip" {
			// 1. Initialize Gzip Reader
			reader, err := gzip.NewReader(bytes.NewReader(bodyBytes))
			if err != nil {
				log.Errorf("Gzip reader init failed: %v", err)
			} else {
				defer func(reader *gzip.Reader) {
					err := reader.Close()
					if err != nil {
						log.Errorf("error closing gzip reader: %v", err)
					}
				}(reader)
				// 2. Decompress the bytes
				decompressed, err := io.ReadAll(reader)
				if err != nil {
					log.Errorf("Decompression failed: %v", err)
				}
				finalBody = decompressed
			}
		} else {
			// Not compressed, use as is
			finalBody = bodyBytes
		}

		// 3. Now Unmarshal the (possibly decompressed) body with indentation
		if len(finalBody) > 0 {
			var indented bytes.Buffer
			if err := json.Indent(&indented, finalBody, "", "  "); err == nil {
				log.Debug("[Pretty-Printed Body]")
				// Using a newline ensures the JSON block starts on its own line
				log.Debugf("\n%s", indented.String())
			} else {
				log.Debugf("[Raw Body]: %s", string(finalBody))
			}
			var getSession GetSession
			err := json.Unmarshal(finalBody, &getSession)
			if err != nil {
				log.Errorf("Unmarshal failed: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			if setAuthJwt != "" {
				c.Cookie(&fiber.Cookie{
					Name:     "comics-galore-jwt",
					Value:    setAuthJwt,
					Expires:  getSession.Session.ExpiresAt, //time.Now().Add(24 * time.Hour),
					HTTPOnly: true,                         // Prevents JS access (Security Best Practice)
					Secure:   true,                         // Only send over HTTPS
					SameSite: "Lax",                        // Recommended for modern browsers
				})
			}
		}
		// --- DEBUG END ---
		return nil
	})
}

func (h *handler) GetUserFromSession(c fiber.Ctx) error {
	// Use a structured logger with the specific operation context
	l := h.logger.With(
		"op", "GetUserFromSession",
		"remote_ip", c.IP(),
	)

	c.Set(fiber.HeaderCacheControl, "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	c.Set(fiber.HeaderPragma, "no-cache")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	// 1. Reconstruct request with context to support cancellation
	targetURL := fmt.Sprintf("%s/api/auth/get-session", h.cfg.Get().BetterAuth)
	req, err := http.NewRequestWithContext(c.Context(), "GET", targetURL, nil)
	if err != nil {
		l.Error("failed to create internal auth request", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create internal auth request")
	}

	// 2. Forward the Cookie header
	// Note: Peek returns a []byte, converting to string is correct.
	cookies := string(c.Request().Header.Peek(h.cfg.Get().SessionKey))
	if cookies == "" {
		l.Debug("no cookies found in request")
		return fiber.NewError(fiber.StatusBadRequest, "no session cookies found in request")
	}
	req.Header.Set(h.cfg.Get().SessionKey, cookies)

	// 3. Execute request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		l.Error("auth service unreachable", "error", err, "url", targetURL)
		return fiber.NewError(fiber.StatusInternalServerError, "auth service unreachable")
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			l.Warn("failed to close response body", "error", closeErr)
		}
	}()

	// 4. Handle Non-200 Status Codes
	if resp.StatusCode != http.StatusOK {
		l.Warn("auth service returned non-200 status", "status", resp.StatusCode)
		return fiber.NewError(fiber.StatusInternalServerError, "auth service returned non-200 status")
	}

	// 5. Decode the User object
	var getSession GetSession

	if err := json.NewDecoder(resp.Body).Decode(&getSession); err != nil {
		l.Error("failed to decode user session data", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to decode user session data")
	}

	l.Debug("user session retrieved", "user_id", getSession.User.Id)
	return c.JSON(getSession.User)
}
