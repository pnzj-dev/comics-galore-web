package auth

import (
	"comics-galore-web/internal/config"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

// HasSession ONLY checks for a session. If found, it populates c.Locals.
// It NEVER blocks the request. Useful for "Guest vs User" header logic.
func HasSession(cfg config.Service, logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenStr := c.Cookies(cfg.Get().SessionKey)
		if tokenStr == "" {
			return c.Next() // No cookie? No problem, move to next handler.
		}

		claims := &BetterAuthClaims{}
		token, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			cfg.Get().JwksFunc.Keyfunc,
			jwt.WithValidMethods([]string{"EdDSA", "RS256", "PS256", "ES256"}),
		)

		// If token is invalid, we don't block, we just don't set the locals
		if err == nil && token.Valid {
			c.Locals("claims", claims)
			logger.Debug("session identified", "user_id", claims.UserID)
		}

		return c.Next()
	}
}

// JWTProtected checks for a session. If missing or invalid, it returns 401.
// Use this for routes that REQUIRE an active user.
func JWTProtected(cfg config.Service, logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenStr := c.Cookies(cfg.Get().SessionKey)

		l := logger.With("ip", c.IP(), "op", "JWTProtected")

		if tokenStr == "" {
			l.Debug("unauthorized: missing session cookie")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Session required"})
		}

		claims := &BetterAuthClaims{}
		token, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			cfg.Get().JwksFunc.Keyfunc,
			jwt.WithValidMethods([]string{"EdDSA", "RS256", "PS256", "ES256"}),
		)

		if err != nil || !token.Valid {
			l.Warn("unauthorized: invalid token", "error", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired session"})
		}

		// Security: Prevent banned users from accessing protected routes
		if claims.Banned {
			l.Info("forbidden: banned user attempt", "user_id", claims.UserID)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Account is banned"})
		}

		c.Locals("claims", claims)
		return c.Next()
	}
}

// HasRole checks if the user in the context has the required role.
func HasRole(allowedRoles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		claims := GetClaims(c)
		if claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
		}

		for _, role := range allowedRoles {
			if claims.Role == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
	}
}
