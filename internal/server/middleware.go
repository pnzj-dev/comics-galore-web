package server

import (
	"comics-galore-web/cmd/web/not_found"
	"comics-galore-web/cmd/web/templates"
	"comics-galore-web/internal/config"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"log"
	"log/slog"
	"time"
)

func (s *FiberServer) setupGlobalMiddleware(cfg config.Service) {
	s.App.Use(s.Timing())
	s.App.Use(s.CustomHeaderMiddleware(cfg))
	s.App.Use(favicon.New(favicon.Config{
		File:         "./public/favicon.ico",
		URL:          "/favicon.ico",
		CacheControl: "public, max-age=31536000",
	}))
	s.App.Use(logger.New(logger.Config{
		Format:     "${cyan}[${time}] ${white}${pid} ${red}${status} ${blue}[${method}] ${white}${path}\n",
		TimeFormat: "02-Jan-2006",
		TimeZone:   "UTC",
	}))
	s.App.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	//s.App.Use(s.NotFound())
	//s.App.Use(auth.SessionCheck(env))
}

func (s *FiberServer) setupRateLimiting(cfg config.Service) {

	l := cfg.GetLogger().With()

	// ────────────────────────────────────────────────
	// Global / catch-all rate limit (very permissive fallback)
	// ────────────────────────────────────────────────
	s.App.Use(limiter.New(limiter.Config{
		Max: 200, // 200 requests
		/*
			Max: func(c fiber.Ctx) int {
				if c.Method() == fiber.MethodPost {
					return 5
				}
				return 100 // GETs are cheap
			},
		*/
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP() // per IP
		},
		/*KeyGenerator: func(c fiber.Ctx) string {
			if c.Method() == "POST" && strings.Contains(c.Path(), "/sign-in") {
				var body struct{ Email string }
				_ = c.Bind().Body(&body)
				if body.Email != "" {
					return body.Email // limit per email instead of IP
				}
			}
			return c.IP()
		},*/
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests, please try again later",
			})
		},
		/*
			LimitReached: func(c fiber.Ctx) error {
				l.Warn("global_rate_limit_hit",
					slog.String("ip", c.IP()),
					slog.String("method", c.Method()),
					slog.String("path", c.Path()),
					slog.String("user_agent", c.Get("User-Agent")),
					slog.Int("limit", 300),
					slog.Duration("window", time.Minute),
				)
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error": "Too many requests – slow down",
				})
			},
		*/
		// Skip trusted proxies/load balancers if needed
		// Skip: func(c fiber.Ctx) bool { return c.IP() == "127.0.0.1" },
	}))

	/*storage, _ := redis.New(redis.Config{
		// your redis conn options
	})

	limiter.New(limiter.Config{
		Storage: storage,
		// ...
	})*/

	// ────────────────────────────────────────────────
	// Stricter limits specifically for /api/v1/auth group
	// ────────────────────────────────────────────────
	authGroup := s.App.Group("/api/v1/auth")

	// Very strict for sign-up (fake account creation is expensive)
	authGroup.Use("/sign-up/*", limiter.New(limiter.Config{
		Max:        5,                // only 5 attempts
		Expiration: 10 * time.Minute, // per 10 min window
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		/*LimitReached: func(c fiber.Ctx) error {
			logger.Warn("rate limit hit on sign-up",
				"ip", c.IP(),
				"path", c.Path(),
			)
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Too many sign-up attempts. Please try again later.",
				"retry_after": "10 minutes",
			})
		},*/
		LimitReached: func(c fiber.Ctx) error {
			l.Warn("rate_limit_hit",
				slog.String("endpoint", "sign-up"),
				slog.String("ip", c.IP()),
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("user_agent", c.Get("User-Agent")),
				slog.Int("max_allowed", 5),
				slog.Duration("window", 10*time.Minute),
				slog.Time("time", time.Now()),
				// Optional: add email from body if you parse it early (advanced)
				// slog.String("email_attempt", parsedEmail),
			)
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Too many sign-up attempts",
				"retry_after": "10 minutes",
			})
		},
	}))

	// Strict for sign-in / password login (credential stuffing protection)
	authGroup.Use("/sign-in/*", limiter.New(limiter.Config{
		Max:        10,
		Expiration: 5 * time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		/*LimitReached: func(c fiber.Ctx) error {
			l.Warn("rate limit hit on sign-in",
				"ip", c.IP(),
				"path", c.Path(),
			)
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Too many login attempts. Please try again later.",
				"retry_after": "5 minutes",
			})
		},*/
		LimitReached: func(c fiber.Ctx) error {
			l.Warn("rate_limit_hit",
				slog.String("endpoint", "sign-in"),
				slog.String("ip", c.IP()),
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("user_agent", c.Get("User-Agent")),
				slog.Int("max_allowed", 10),
				slog.Duration("window", 5*time.Minute),
			)
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Too many login attempts",
				"retry_after": "5 minutes",
			})
		},
	}))

	// Medium limit for password reset / email verification attempts
	authGroup.Use("/reset-password/*", limiter.New(limiter.Config{
		Max:        3,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		/*LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many password reset requests.",
			})
		},*/
		LimitReached: func(c fiber.Ctx) error {
			l.Warn("rate_limit_hit",
				slog.String("endpoint", "reset-password"),
				slog.String("ip", c.IP()),
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("user_agent", c.Get("User-Agent")),
				slog.Int("max_allowed", 3),
				slog.Duration("window", 15*time.Minute),
			)
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many password reset requests",
			})
		},
	}))

	// Optional: more permissive for OAuth/social sign-in flows
	// (or skip rate limiting entirely for GET /oauth/*)
	// authGroup.Use(limiter.New(limiter.Config{ ... })) // or no limiter here
}

func (s *FiberServer) CustomHeaderMiddleware(cfg config.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("X-App-Version", cfg.Get().Version)
		return c.Next()
	}
}

// Timing auth measures the duration of the request and appends the result to the response headers.
// It helps track request processing times for performance analysis.
func (s *FiberServer) Timing() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		// Process the request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Format duration in milliseconds for the header
		// "app;dur=12.5" is the standard format
		durMs := float64(duration.Nanoseconds()) / 1e6
		c.Append("Server-Timing", fmt.Sprintf("app;dur=%.2f", durMs))
		log.Printf("%s %s processed in %v", c.Method(), c.Path(), duration)
		return err
	}
}

// NotFound handles 404 errors by rendering a templ component.
// This should be registered as the last handler in your Fiber app.
func (s *FiberServer) NotFound() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Set the status to 404
		c.Status(fiber.StatusNotFound)
		log.Printf("404 Not Found: %s", c.OriginalURL())

		// For API requests, you might still want to return JSON
		if c.Accepts("json") != "" && c.Accepts("html") == "" {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Resource not found",
				"path":    c.OriginalURL(),
			})
		}

		title := "404 - Page Not Found | Comics Galore"
		notFound := not_found.NotFound(c.OriginalURL())
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return templates.BasicLayout(title, notFound).Render(c.Context(), c.Response().BodyWriter())
	}
}
