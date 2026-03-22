package auth

import (
	"comics-galore-web/cmd/web/views/partials/messages"
	"comics-galore-web/cmd/web/views/partials/modals"
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"
)

func (h *handler) handleAuthResponse(c fiber.Ctx) error {
	if err := c.Next(); err != nil {
		return err
	}

	if c.Get("HX-Request") == "true" {
		statusCode := c.Response().StatusCode()
		form, _ := c.Locals("form").(string)

		c.Response().Header.Del(fiber.HeaderContentEncoding)
		c.Response().Header.Del(fiber.HeaderContentLength)
		c.Response().ResetBody()

		if statusCode >= 400 {
			c.Set("HX-Retarget", "#form-error")
			c.Set("HX-Reswap", "innerHTML")
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
			c.Status(fiber.StatusOK)
			return messages.AuthError(statusCode).Render(c.Context(), c.Response().BodyWriter())
		}

		c.Set("HX-Trigger", form+"-success") // e.g., "login-success" or "reset-password-success"
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
		c.Status(fiber.StatusOK)
		return modals.Success(form).Render(c.Context(), c.Response().BodyWriter())
	}
	return nil
}

func (h *handler) renderError(c fiber.Ctx, formName string, errs map[string]string) error {
	siteKey := h.cfg.Get().Cloudflare.TurnstileSiteKey
	var component templ.Component

	switch formName {
	case "sign-in":
		component = modals.Signin(siteKey, errs)
	case "sign-up":
		component = modals.Signup(siteKey, errs)
	case "reset-password":
		component = modals.ResetPassword(siteKey, errs)
	default:
		component = modals.Error("Request Failed", "An unexpected error occurred.")
	}
	return h.renderComponent(c, component)
}

func (h *handler) renderComponent(c fiber.Ctx, component templ.Component) error {
	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "authError")
		c.Set("HX-Retarget", "#form-error")
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
		c.Status(fiber.StatusOK)
		return component.Render(c.Context(), c.Response().BodyWriter())
	}
	c.Status(fiber.StatusUnprocessableEntity)
	return component.Render(c.Context(), c.Response().BodyWriter())
}
