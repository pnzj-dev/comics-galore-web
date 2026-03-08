package auth

import (
	"bytes"
	"comics-galore-web/cmd/web/auth2"
	authz "comics-galore-web/internal/auth2"
	"comics-galore-web/internal/cloudflare"
	"comics-galore-web/internal/config"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
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
	cfg       config.Service
	logger    *slog.Logger
	turnstile cloudflare.Turnstile
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
	GetUserFromSession(c fiber.Ctx) error
}

func NewHandler(cfg config.Service, turnstile cloudflare.Turnstile) Handler {
	return &handler{
		cfg:       cfg,
		turnstile: turnstile,
		logger:    cfg.GetLogger().With("component", "auth_proxy"),
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

		/**/

		// Only protect POST requests (sign-up, sign-in, reset password, etc.)
		// GETs like /session, OAuth redirects usually don't need captcha
		if c.Method() == fiber.MethodPost {

			//binding

			var input any
			var tab string // for error rendering

			switch {
			case strings.HasPrefix(path, "sign-in"):
				input = new(authz.LoginInput)
				tab = "login"
			case strings.HasPrefix(path, "sign-up"):
				input = new(authz.SignupInput)
				tab = "signup"
			case strings.HasPrefix(path, "reset-password"):
				input = new(authz.ForgotInput)
				tab = "forgot"
			default:
				// other POSTs → no early validation
				goto proxy
			}

			// 1. Bind & validate input
			if err := c.Bind().Body(input); err != nil {
				// Fiber v3 returns validator.ValidationErrors or other bind errors
				errorsMap := make(map[string]string)
				if ve, ok := err.(validator.ValidationErrors); ok {
					for _, fe := range ve {
						errorsMap[fe.Field()] = fe.Tag() // or use translator for better messages
						// Better: use universal translator for human messages (see below)
					}
				} else {
					errorsMap["general"] = "Invalid request format"
				}

				return renderErrorTab(c, tab, errorsMap, c.FormValue("email"), h.cfg.Get().Cloudflare.TurnstileSiteKey)
			}

			token := c.Get("x-captcha-response") // ← Header sent by frontend
			remoteIP := c.IP()                   // Or use X-Forwarded-For if behind proxy

			if token == "" {
				h.logger.Warn("turnstile token missing on POST request",
					"path", path,
					"remote_ip", remoteIP,
				)
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "captcha token required",
				})
			}

			// Verify with your turnstile service (from earlier code)
			success, err := h.turnstile.Verify(c.Context(), token, h.cfg.Get().Cloudflare.TurnstileSecretKey, remoteIP)
			if err != nil || !success.Success {
				h.logger.Warn("turnstile verification failed",
					"error", err,
					"path", path,
					"token_prefix", token[:min(12, len(token))]+"...",
					"remote_ip", remoteIP,
				)
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "captcha verification failed",
				})
			}

			h.logger.Debug("turnstile verified successfully",
				"path", path,
				"remote_ip", remoteIP,
			)

			// Forward the token header to Better Auth (required by its captcha plugin)
			//c.Request().Header.Set("x-captcha-response", token)
			// Optional: also forward IP if your Better Auth config uses it
			//c.Request().Header.Set("x-captcha-user-remote-ip", remoteIP)
		}
	proxy:

		/**/

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

		// --- HTMX SUPPORT ---
		// Inside the group.All("/*", func(c fiber.Ctx) error { ... })
		if c.Get("HX-Request") == "true" {
			// If Better Auth returned error (4xx/5xx) or we have Turnstile error
			if c.Response().StatusCode() >= 400 || c.Locals("turnstileError") != nil {
				// Reuse the same VM + templ.KV pattern
				errors := map[string]string{
					"general": "Invalid credentials or captcha failed", // map from Better Auth JSON or your Turnstile error
				}
				vm := auth2.AuthModalVM{
					Tab:              detectTabFromPath(path), // helper
					Errors:           errors,
					Values:           extractValuesFromRequest(c), // optional repopulation
					TurnstileSiteKey: h.cfg.Get().Cloudflare.TurnstileSiteKey,
				}
				c.Response().Header.Set("Content-Type", "text/html")
				return auth2.AuthTabContent(vm).Render(c.Context(), c.Response().BodyWriter())
			}
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

func renderErrorTab(c fiber.Ctx, tab string, errors map[string]string, email string, siteKey string) error {
	values := map[string]string{"email": email} // repopulate email if present

	vm := auth2.AuthModalVM{
		Tab:              tab,
		Errors:           errors,
		Values:           values,
		TurnstileSiteKey: siteKey,
	}

	c.Set("Content-Type", "text/html")
	return auth2.AuthTabContent(vm).Render(c.Context(), c.Response().BodyWriter())
}
