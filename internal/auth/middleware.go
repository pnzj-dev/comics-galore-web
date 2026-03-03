package auth

import (
	"comics-galore-web/internal/config"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// HasSession ONLY checks for a session. If found, it populates c.Locals.
// It NEVER blocks the request. Useful for "Guest vs User" header logic.
import (
	"github.com/gofiber/fiber/v3/extractors"
)

func HasSession(cfg config.Service) fiber.Handler {
	/*
		jwtExtractor := extractors.Chain(
			extractors.FromHeader("Authorization"), // Try header first
			extractors.FromCookie("jwt_token"),      // Fallback to cookie
		)
	*/

	return func(c fiber.Ctx) error {
		jwtExtractor := extractors.FromCookie("comics-galore-jwt")

		// 1. Use the extractor to get the token
		tokenStr, err := jwtExtractor.Extract(c)

		// Extractors in v3 return an error if the key is missing
		if err != nil || tokenStr == "" {
			return c.Next()
		}

		var claims Claims
		token, err := jwt.ParseWithClaims(
			tokenStr,
			&claims,
			cfg.Get().JwksFunc.Keyfunc,
			jwt.WithValidMethods([]string{"EdDSA", "RS256", "PS256", "ES256"}),
			//jwt.WithIssuer("https://comics-galore-auth-staging.amadioha.workers.dev"),
			//jwt.WithAudience("https://comics-galore-auth-staging.amadioha.workers.dev"),
			jwt.WithoutClaimsValidation(),
			jwt.WithLeeway(5*time.Second),
		)

		if err != nil {
			cfg.GetLogger().Error("jwt_parsing_failed", "error", err)
			return c.Next()
		}

		if token != nil && token.Valid {
			c.Locals("claims", claims)
			cfg.GetLogger().Debug("session identified", "user_id", claims.Id)
		}

		return c.Next()
	}
}

// JWTProtected checks for a session. If missing or invalid, it returns 401.
// Use this for routes that REQUIRE an active user.
func JWTProtected(cfg config.Service) fiber.Handler {
	// 1. Initialize extractor once outside the request handler
	jwtExtractor := extractors.FromCookie("jwt")

	// 2. Base logger (shared across requests)
	baseLogger := cfg.GetLogger().With("op", "JWTProtected")

	return func(c fiber.Ctx) error {
		// 3. Extract token using v3 extractor
		tokenStr, err := jwtExtractor.Extract(c)

		l := baseLogger.With("ip", c.IP())

		if err != nil || tokenStr == "" {
			l.Debug("unauthorized: missing session cookie")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Session required",
			})
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			cfg.Get().JwksFunc.Keyfunc,
			jwt.WithValidMethods([]string{"EdDSA", "RS256", "PS256", "ES256"}),
			jwt.WithLeeway(5*time.Second),
		)

		// 4. Validate Token & Signature
		if err != nil || token == nil || !token.Valid {
			l.Warn("unauthorized: invalid token", "error", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired session",
			})
		}

		// 5. Business Logic: Check Banned Status
		if claims.Banned {
			l.Info("forbidden: banned user attempt", "user_id", claims.Id)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Account is banned",
			})
		}

		// 6. Persist claims for the rest of the request chain
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
