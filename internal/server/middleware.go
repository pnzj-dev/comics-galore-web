package server

import (
	"comics-galore-web/cmd/web"
	"comics-galore-web/cmd/web/templates"
	"comics-galore-web/internal/config"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"log"

	"time"
)

func (s *FiberServer) setupGlobalMiddleware(cfg config.Service) {
	s.App.Use(s.Timing())
	s.App.Use(s.CustomHeaderMiddleware())
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

func (s *FiberServer) CustomHeaderMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("X-App-Version", s.config.Get().Version)
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
		notFound := web.NotFound(c.OriginalURL())
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return templates.BasicLayout(title, notFound).Render(c.Context(), c.Response().BodyWriter())
	}
}
