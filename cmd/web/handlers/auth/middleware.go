package auth

import (
	"comics-galore-web/internal/auth"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"strings"
)

// Now it returns a fiber.Handler (which is func(fiber.Ctx) error)
func (h *handler) validateInput() fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Method() != fiber.MethodPost {
			return c.Next()
		}
		if c.Request().Header.ContentLength() <= 0 {
			h.logger.Warn("no content")
			return c.Next()
		}
		path := c.Path()
		var formInput any
		var templateName string

		// 1. Identify context based on Path
		switch {
		case strings.HasSuffix(path, "/sign-in/email"):
			formInput = new(auth.LoginInput)
			templateName = "sign-in"
			c.Locals("form", templateName)
			c.Locals("formInput", formInput)
			break
		case strings.HasSuffix(path, "/sign-up/email"):
			formInput = new(auth.SignupInput)
			templateName = "sign-up"
			c.Locals("form", templateName)
			c.Locals("formInput", formInput)
			break
		case strings.HasSuffix(path, "/reset-password"):
			formInput = new(auth.ForgotInput)
			templateName = "reset-password"
			c.Locals("form", templateName)
			c.Locals("formInput", formInput)
			break
		default:
			return c.Next()
		}

		// 2. Bind the body
		if err := c.Bind().Body(formInput); err != nil {
			h.logger.Warn("invalid input format", "path", path, "error", err)
			return h.renderError(c, templateName, map[string]string{"general": "Invalid input format"})
		}

		// 3. Structural Validation (using go-playground/validator)
		// Assuming h.validate is initialized as validator.New()
		if err := h.validate.Struct(formInput); err != nil {
			errors := make(map[string]string)
			// Map validator errors to user-friendly messages
			for _, err := range err.(validator.ValidationErrors) {
				field := strings.ToLower(err.Field())
				errors[field] = fmt.Sprintf("Invalid %s provided", field)
			}
			return h.renderError(c, templateName, errors)
		}
		return c.Next()
	}
}

func (h *handler) withCookieSync() fiber.Handler {
	return func(c fiber.Ctx) error {
		// 1. Let the request continue to the next handler (the Proxy)
		if err := c.Next(); err != nil {
			return err
		}

		// 2. After the Proxy has finished and populated c.Response()
		// we check if we need to sync cookies
		if err := h.syncCookies(c); err != nil {
			h.logger.Error("failed to sync auth cookies", "error", err)
			// We don't return the error to the user here because the
			// main auth action might have actually succeeded.
		}

		return nil
	}
}
