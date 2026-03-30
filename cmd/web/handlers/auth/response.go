package auth

import (
	"comics-galore-web/cmd/web/views/partials/modals"
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"strconv"
	"time"
)

func (h *handler) manageResponse(c fiber.Ctx) error {
	// 1. Wait for everything down-stack (including createCookie and Proxy)
	if err := c.Next(); err != nil {
		return err
	}

	// 1. Get all raw Set-Cookie header values from the upstream response
	setCookies := c.Response().Header.PeekAll("Set-Cookie")

	for i, cookie := range setCookies {
		if len(cookie) > 0 {
			// 2. Forward each one to the client's browser
			c.Append("Set-Cookie", string(cookie))
			log.Infof("manageResponse => Cookie [%d]: %s", i, string(cookie))
		}
	}

	if len(setCookies) > 0 {
		log.Debugf("Proxy: Forwarded %d cookies to client", len(setCookies))
	}

	// 2. Capture the original status from the backend/proxy
	statusCode := c.Response().StatusCode()
	formType := h.getFormTypeFromPath(c.Path())

	// 3. Prepare the response for an HTMX fragment swap
	// We keep Set-Cookie but clear the body and encoding
	c.Response().Header.Del(fiber.HeaderContentEncoding)
	c.Response().Header.Del(fiber.HeaderContentLength)
	c.Response().ResetBody()

	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	c.Status(fiber.StatusOK) // Always 200 for HTMX swaps

	// 4. Handle Errors
	if statusCode >= 400 {
		return h.renderError(c, formType, errorMessage(statusCode))
	}

	if formType == "sign-out" {
		c.Cookie(&fiber.Cookie{
			Name:     "cg-auth-local",
			Value:    "",
			Expires:  time.Now().Add(-time.Hour),
			HTTPOnly: true,
		})
	}

	// 5. Handle Success
	c.Set("HX-Trigger", "auth-success")

	// Use h.renderComponent or similar to ensure the writer is handled
	return modals.Success(formType).Render(c.Context(), c.Response().BodyWriter())
}

func (h *handler) renderError(c fiber.Ctx, formType string, errs map[string]string) error {
	var component templ.Component
	//formType := h.getFormTypeFromPath(c.Path())

	switch formType {
	case "sign-in":
		component = modals.Signin(errs)
	case "sign-up":
		component = modals.Signup(errs)
	case "reset-password":
		component = modals.ResetPassword(errs)
	default:
		component = modals.Error("Request Failed", "An unexpected error occurred.")
	}

	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	c.Status(fiber.StatusOK)
	return templ.RenderFragments(c.Context(), c.Response().BodyWriter(), component, formType)
}

func errorMessage(statusCode int) map[string]string {
	var msg string
	switch statusCode {
	case 401:
		msg = "Invalid email or password. Please try again."
	case 403:
		msg = "Your account has been disabled. Please contact support."
	case 404:
		msg = "No account found with that email address."
	case 422:
		msg = "Validation failed. Please check your inputs."
	case 429:
		msg = "Too many attempts. Please wait a moment before trying again."
	case 500:
		msg = "Our servers are having trouble. Please try again later."
	default:
		msg = "An unexpected error occurred (Status: " + strconv.Itoa(statusCode) + ")"
	}
	return map[string]string{"general": msg}
}
