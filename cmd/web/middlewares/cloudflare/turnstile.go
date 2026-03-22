package cloudflare

import (
	"comics-galore-web/internal/cloudflare"
	"comics-galore-web/internal/config"
	"github.com/gofiber/fiber/v3"
)

func VerifyTurnstile(svc cloudflare.Turnstile, cfg config.Service, render func(c fiber.Ctx, tab string, errs map[string]string) error) fiber.Handler {
	return func(c fiber.Ctx) error {

		l := cfg.GetLogger().With()

		token := c.Get("x-captcha-response")
		remoteIP := c.IP()

		if cfg.Get().AppEnv == "development" || token == "disabled-token" {
			cfg.GetLogger().Debug("Turnstile bypassed in development mode")
			return c.Next()
		}

		// 1. Define a helper for uniform error reporting
		reportError := func(msg string, fieldErr string) error {
			form, isHTMX := c.Locals("form").(string), c.Get("HX-Request") == "true"

			if isHTMX && form != "" {
				return render(c, form, map[string]string{"captcha": fieldErr})
			}
			return fiber.NewError(fiber.StatusUnauthorized, "Turnstile: "+msg)
		}

		// 2. Check for token existence
		if token == "" {
			l.Warn("turnstile token missing", "remote_ip", remoteIP)
			return reportError("captcha required", "Required")
		}

		// 3. Perform verification
		success, err := svc.Verify(
			c.Context(),
			token,
			cfg.Get().Cloudflare.TurnstileSecretKey,
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
}
