package auth

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
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

	err := proxy.Do(c, fullTarget)

	// Prevent sensitive auth data from being cached by browsers/CDNs
	c.Set(fiber.HeaderCacheControl, "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	c.Set(fiber.HeaderPragma, "no-cache")

	// Security hardening
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set("X-Frame-Options", "DENY") // Prevents clickjacking

	return err
}
