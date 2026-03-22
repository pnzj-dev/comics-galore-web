package pages

import (
	"comics-galore-web/cmd/web/views/helper"
	"comics-galore-web/internal/auth"
	"comics-galore-web/internal/view"
	"fmt"
	"github.com/gofiber/fiber/v3"
)

func fullTitle(title, env string) string {
	if env != "production" && env != "prd" && env != "prod" {
		return fmt.Sprintf("Comics-Galore - %s", env)
	}
	return title
}

func (h *handler) ViewMiddleware(c fiber.Ctx) error {
	s := view.AppContext{
		Title:             fullTitle("Comics Galore", h.cfg.Get().AppEnv),
		TurnstileSiteKey:  h.cfg.Get().Cloudflare.TurnstileSiteKey,
		TurnstileEnabled:  false,
		Claims:            auth.GetClaims(c),
		Variants:          map[string]string{"view": "public", "preview": "cover", "thumbnail": "thumbnail"},
		Tags:              []view.Tag{{0, "tag1", "tag1"}},
		Categories:        []view.Category{{0, "category1", "category1"}},
		SubscriptionPlans: []view.SubscriptionPlan{{0, "name1", 10.00, 10, 0.10}},
	}
	c.SetContext(helper.WithAppContext(c.Context(), &s))
	return c.Next()
}
