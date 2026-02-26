package auth

import (
	"comics-galore-web/internal/config"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
)

type handler struct {
	targetURL string
	logger    *slog.Logger
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
	GetUserFromSession(c fiber.Ctx) error
}

func NewHandler(cfg config.Service, logger *slog.Logger) Handler {
	return &handler{
		targetURL: cfg.Get().BetterAuth,
		logger:    logger.With("component", "auth_proxy"),
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	// Clean the base URL once during registration
	targetBase := strings.TrimSuffix(h.targetURL, "/")

	h.logger.Info("registering auth proxy route", "target", targetBase)

	group := app.Group("/api/v1/auth")
	group.Get("/get-session", h.GetUserFromSession)

	app.All("/api/auth/*", func(c fiber.Ctx) error {
		// 1. Capture the wildcard
		remainingPath := c.Params("*")

		// 2. Reconstruct URL carefully
		// We use c.OriginalURL() or manually append the Query String to ensure
		// OAuth redirects and callbacks (which use heavy query params) don't break.
		fullTarget := fmt.Sprintf("%s/%s", targetBase, remainingPath)
		if queryString := string(c.Request().URI().QueryString()); queryString != "" {
			fullTarget = fmt.Sprintf("%s?%s", fullTarget, queryString)
		}

		l := h.logger.With(
			"method", c.Method(),
			"path", c.Path(),
			"proxy_to", fullTarget,
		)

		l.Debug("proxying request")

		// 3. Execute Proxy
		if err := proxy.Do(c, fullTarget); err != nil {
			l.Error("proxy request failed", "error", err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"error": "authentication service unreachable",
			})
		}

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
	// BUG FIX: Using the request context ensures that if the user hangs up,
	// the internal auth request is also cancelled.
	targetURL := fmt.Sprintf("%s/api/auth/get-session", h.targetURL)
	req, err := http.NewRequestWithContext(c.Context(), "GET", targetURL, nil)
	if err != nil {
		l.Error("failed to create internal auth request", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create internal auth request")
	}

	// 2. Forward the Cookie header
	// Note: Peek returns a []byte, converting to string is correct.
	cookies := string(c.Request().Header.Peek("Cookie"))
	if cookies == "" {
		l.Debug("no cookies found in request")
		return fiber.NewError(fiber.StatusBadRequest, "no session cookies found in request")
	}
	req.Header.Set("Cookie", cookies)

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
	// BUG FIX: Better-Auth will return 401 if the session is invalid.
	// Decoding a 401 body usually results in an empty User struct.
	if resp.StatusCode != http.StatusOK {
		l.Warn("auth service returned non-200 status", "status", resp.StatusCode)
		return fiber.NewError(fiber.StatusInternalServerError, "auth service returned non-200 status")
	}

	// 5. Decode the User object
	var sessionData struct {
		User User `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sessionData); err != nil {
		l.Error("failed to decode user session data", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to decode user session data")
	}

	l.Debug("user session retrieved", "user_id", sessionData.User.ID)
	return c.JSON(sessionData.User)
}
