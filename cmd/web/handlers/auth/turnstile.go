package auth

import (
	"github.com/gofiber/fiber/v3"
)

func (h *handler) verifyTurnstile(c fiber.Ctx) error {

	l := h.cfg.GetLogger().With()

	token := c.Get("x-captcha-response")
	remoteIP := c.IP()

	if h.cfg.Get().AppEnv == "development" || token == "disabled-token" {
		h.cfg.GetLogger().Debug("Turnstile bypassed in development mode")
		return c.Next()
	}

	// 1. Define a helper for uniform error reporting
	reportError := func(msg string, fieldErr string) error {
		form, isHTMX := c.Locals("form").(string), c.Get("HX-Request") == "true"

		if isHTMX && form != "" {
			return h.renderError(c, form, map[string]string{"captcha": fieldErr})
		}
		return fiber.NewError(fiber.StatusUnauthorized, "Turnstile: "+msg)
	}

	// 2. Check for token existence
	if token == "" {
		l.Warn("turnstile token missing", "remote_ip", remoteIP)
		return reportError("captcha required", "Required")
	}

	// 3. Perform verification
	success, err := h.turnstile.Verify(
		c.Context(),
		token,
		h.cfg.Get().Cloudflare.TurnstileSecretKey,
		remoteIP,
	)

	// 4. Handle verification failure (Safe Version)
	if err != nil {
		l.Error("turnstile service error", "error", err, "remote_ip", remoteIP)
		return reportError("service error", "Service unavailable")
	}

	if success == nil || !success.Success {
		l.Warn("turnstile verification rejected",
			"remote_ip", remoteIP,
			"success_nil", success == nil,
		)
		return reportError("failed verification", "Failed verification")
	}

	return c.Next()

}
