package auth

import (
	"comics-galore-web/internal/auth"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

func (h *handler) validateInput(c fiber.Ctx) error {
	if c.Method() != fiber.MethodPost {
		return c.Next()
	}

	if c.Request().Header.ContentLength() <= 0 {
		h.logger.Warn("no content")
		return c.Next()
	}

	var formInput any
	formType := h.getFormTypeFromPath(c.Path())

	switch formType {
	case "sign-in":
		formInput = new(auth.LoginInput)
		c.Locals("formInput", formInput)
	case "sign-up":
		formInput = new(auth.SignupInput)
		c.Locals("formInput", formInput)
	case "reset-password":
		formInput = new(auth.ForgotInput)
		c.Locals("formInput", formInput)
	default:
		return c.Next()
	}

	// 2. Bind the body
	if err := c.Bind().Body(formInput); err != nil {
		h.logger.Warn("invalid input format", "path", c.Path(), "error", err)
		return h.renderError(c, formType, map[string]string{"general": "Invalid input format"})
	}

	// 3. Structural Validation (using go-playground/validator)
	if err := h.validate.Struct(formInput); err != nil {
		errors := make(map[string]string)
		// Map validator errors to user-friendly messages
		for _, err := range err.(validator.ValidationErrors) {
			errors[err.Field()] = fmt.Sprintf("Invalid %s provided", err.Field())
		}
		return h.renderError(c, formType, errors)
	}
	return c.Next()

}

func (h *handler) headerDebugMiddleware(c fiber.Ctx) error {
	// 1. Let the request continue to your handler/proxy
	err := c.Next()

	// 2. After the handler finishes, inspect the response headers
	log.Info("--- Outgoing Header Debug ---")
	log.Infof("Status: %d", c.Response().StatusCode())

	// PeekAll returns a [][]byte of all headers with this key
	setCookies := c.Response().Header.PeekAll("Set-Cookie")

	if len(setCookies) == 0 {
		log.Warn("No Set-Cookie headers found in this response.")
	} else {
		for i, cookie := range setCookies {
			//c.Append("Set-Cookie", string(cookie))
			log.Infof("Cookie [%d]: %s", i, string(cookie))
		}
	}
	log.Info("-----------------------------")

	return err
}
