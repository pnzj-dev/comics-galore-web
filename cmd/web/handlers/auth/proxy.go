package auth

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"strings"
)

func (h *handler) proxyToBetterAuth(c fiber.Ctx) error {
	targetBase := strings.TrimSuffix(h.cfg.Get().BetterAuth, "/")

	// 1. Capture everything after /api/v1/auth/
	// If request is /api/v1/auth/sign-in/email, path is "sign-in/email"
	path := strings.TrimPrefix(c.Params("*"), "/")

	// 2. Construct final destination URL
	fullTarget := fmt.Sprintf("%s/%s", targetBase, path)

	body := c.Body() // Get the body sent by HTMX

	// --- DEBUG LOGGING START ---
	log.Info("--- Proxy Request to Better-Auth ---")
	log.Infof("URL: %s", fullTarget)
	log.Infof("Method: POST")
	log.Infof("Content-Type Sent: %s", c.Get("Content-Type"))
	log.Infof("Content-Length: %d", len(body))
	log.Infof("Body Content: %s", string(body))

	// Check if the Cookie is actually being forwarded
	cookie := c.Get("Cookie")
	if cookie != "" {
		log.Infof("Cookie Present: Yes (Length: %d)", len(cookie))
	} else {
		log.Warn("Cookie Present: NO (Sign-out will likely fail 401)")
	}
	log.Info("------------------------------------")
	// --- DEBUG LOGGING END ---

	// 3. Preserve Query Parameters (Essential for OAuth callbacks and email verification)
	if qs := string(c.Request().URI().QueryString()); qs != "" {
		fullTarget = fmt.Sprintf("%s?%s", fullTarget, qs)
	}

	// Set internal Auth headers
	c.Request().Header.Set(fiber.HeaderAuthorization, "Bearer "+h.cfg.Get().BetterAuthSecret)
	c.Request().Header.SetHost(strings.TrimPrefix(strings.TrimPrefix(targetBase, "http://"), "https://"))

	// Manage Origin for CORS
	origin := c.Get("Origin")
	if origin == "" {
		origin = "localhost"
	}
	c.Request().Header.Set("Origin", origin)

	// CRITICAL: Manually copy headers to ensure Content-Type is application/json
	/*c.Request().Header.Set("Content-Type", "application/json")
	if cookie != "" {
		c.Request().Header.Set("Cookie", cookie)
	}*/

	err := proxy.Do(c, fullTarget)

	// Forward response back to browser (as we discussed before)
	setCookies := c.Response().Header.Cookies()
	for _, ck := range setCookies {
		c.Append("Set-Cookie", string(ck))
	}

	// Prevent sensitive auth data from being cached by browsers/CDNs
	c.Set(fiber.HeaderCacheControl, "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	c.Set(fiber.HeaderPragma, "no-cache")

	// Security hardening
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set("X-Frame-Options", "DENY") // Prevents clickjacking

	return err
}
