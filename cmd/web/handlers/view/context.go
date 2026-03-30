package view

import (
	"comics-galore-web/internal/config"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

func fullTitle(title, env string) string {
	if env != "production" && env != "prd" && env != "prod" {
		return fmt.Sprintf("Comics-Galore - %s", env)
	}
	return title
}

func GetAppContext(env *config.Env) fiber.Handler {
	return func(c fiber.Ctx) error {
		userInfo := GetClaims(c)

		// 1. Log the incoming UserInfo state
		if userInfo == nil {
			log.Debug("GetAppContext: No UserInfo found (Guest Session)")
		} else {
			log.Debugf("GetAppContext: UserInfo found for ID: %s", userInfo.UserID)
		}

		s := AppContext{
			Title:             fullTitle("Comics Galore", env.AppEnv),
			TurnstileSiteKey:  env.Cloudflare.TurnstileSiteKey,
			TurnstileEnabled:  false,
			UserInfo:          userInfo,
			Variants:          map[string]string{"view": "public", "preview": "cover", "thumbnail": "thumbnail"},
			Tags:              []Tag{{0, "tag1", "tag1"}},
			Categories:        []Category{{0, "category1", "category1"}},
			SubscriptionPlans: []SubscriptionPlan{{0, "name1", 10.00, 10, 0.10}},
		}

		// 2. Attach to context and log the address
		newCtx := WithAppContext(c.Context(), &s)
		c.SetContext(newCtx)

		log.Infof("GetAppContext: AppContext attached to request. Pointer: %p, User: %v", &s, s.UserInfo != nil)

		return c.Next()
	}
}

/*if userInfo != nil {
	// Indent with 2 spaces for readability
	prettyJSON, _ := json.MarshalIndent(userInfo, "", "  ")
	log.Infof("Claims Structure:\n%s", string(prettyJSON))
}*/
