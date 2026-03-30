package auth

import (
	"comics-galore-web/cmd/web/handlers/view"
	"comics-galore-web/internal/config"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
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
		jwtExtractor := extractors.FromCookie("cg-auth-local")

		// 1. Use the extractor to get the token
		tokenStr, err := jwtExtractor.Extract(c)

		// Extractors in v3 return an error if the key is missing
		if err != nil || tokenStr == "" {
			return c.Next()
		}

		var userInfo UserInfo
		token, err := jwt.ParseWithClaims(
			tokenStr,
			&userInfo,
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
			c.Locals("userInfo", userInfo)
			cfg.GetLogger().Debug("session identified", "user_id", userInfo.ID)
		}

		return c.Next()
	}
}

func SessionLoader(cfg config.Service) fiber.Handler {
	// Get the logger once from config
	logger := cfg.GetLogger()

	return func(c fiber.Ctx) error {
		tokenStr := c.Cookies("cg-auth-local")

		log.Infof("Cookie string => %s", tokenStr)

		if tokenStr == "" {
			c.Locals("session_loaded", false)
			return c.Next()
		}

		var claims view.ComicsGaloreClaims
		token, err := jwt.ParseWithClaims(
			tokenStr,
			&claims,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(cfg.Get().JwtSecret), nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithLeeway(60*time.Second),
		)

		// 2. Structured Error Logging
		if err != nil {
			logger.Warn("jwt_parsing_failed",
				slog.String("error", err.Error()),
				slog.String("path", c.Path()),
				slog.String("ip", c.IP()),
				slog.String("user_agent", c.Get("User-Agent")),
			)
			c.Locals("session_loaded", false)
			return c.Next()
		}

		// 3. Structured Success Logging
		if token != nil && token.Valid {
			c.Locals("claims", &claims)

			c.Locals("userID", claims.UserID)
			c.Locals("session_loaded", true)

			logger.Debug("session_identified",
				slog.String("user_id", claims.UserID),
				slog.String("email", claims.Email),
				slog.String("role", claims.Role),
			)
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

		userInfo := &UserInfo{}
		token, err := jwt.ParseWithClaims(
			tokenStr,
			userInfo,
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
		if userInfo.Banned {
			l.Info("forbidden: banned user attempt", "user_id", userInfo.ID)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Account is banned",
			})
		}

		// 6. Persist userInfo for the rest of the request chain
		c.Locals("userInfo", userInfo)
		return c.Next()
	}
}

// HasRole checks if the user in the context has the required role.
func HasRole(allowedRoles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userInfo := view.GetClaims(c)
		if userInfo == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
		}

		for _, role := range allowedRoles {
			if userInfo.Role == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
	}
}
