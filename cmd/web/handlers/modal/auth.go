package modal

import (
	"comics-galore-web/cmd/web/views/partials/modals"
	"github.com/gofiber/fiber/v3"
)

func (h *handler) showSigninModal(c fiber.Ctx) error {
	return modals.Signin(h.cfg.Get().Cloudflare.TurnstileSiteKey, map[string]string{}).Render(c, c.Response().BodyWriter())
}

func (h *handler) showSignupModal(c fiber.Ctx) error {
	return modals.Signup(h.cfg.Get().Cloudflare.TurnstileSiteKey, map[string]string{}).Render(c, c.Response().BodyWriter())
}

func (h *handler) showForgotModal(c fiber.Ctx) error {
	return modals.ResetPassword(h.cfg.Get().Cloudflare.TurnstileSiteKey, map[string]string{}).Render(c, c.Response().BodyWriter())
}

func (h *handler) showLogoutModal(c fiber.Ctx) error {
	return modals.Logout().Render(c, c.Response().BodyWriter())
}
