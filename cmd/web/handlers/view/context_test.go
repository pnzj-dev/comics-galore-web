package view

import (
	"github.com/golang-jwt/jwt/v5"
	"net/http/httptest"
	"testing"
	"time"

	"comics-galore-web/internal/config"
	"github.com/gofiber/fiber/v3"
)

func TestGetAppContextMiddleware(t *testing.T) {
	// 1. Setup mock environment

	env := &config.Env{
		AppEnv:    "development",
		JwtSecret: "test-32-character-secret-key-12345",
		Cloudflare: struct {
			ImagesAPIKey       string `env:"CLOUDFLARE_IMAGES_APIKEY"`
			AccountID          string `env:"CLOUDFLARE_IMAGES_ACCOUNT_ID"`
			ImagesURL          string `env:"CLOUDFLARE_IMAGES_IMAGES_URL"`
			R2Bucket           string `env:"CLOUDFLARE_R2_BUCKET"`
			R2Endpoint         string `env:"CLOUDFLARE_R2_ENDPOINT"`
			R2AccessKey        string `env:"CLOUDFLARE_R2_ACCESS_KEY"`
			R2SecretKey        string `env:"CLOUDFLARE_R2_SECRET_ACCESS_KEY"`
			TurnstileURL       string `env:"CLOUDFLARE_TURNSTILE_URL"`
			TurnstileSiteKey   string `env:"CLOUDFLARE_TURNSTILE_SITE_KEY"`
			TurnstileSecretKey string `env:"CLOUDFLARE_TURNSTILE_SECRET_KEY"`
		}{
			ImagesAPIKey:       "images-api-key",
			AccountID:          "account-id",
			ImagesURL:          "images-url",
			R2Bucket:           "r2-bucket",
			R2Endpoint:         "r2-endpoint",
			R2AccessKey:        "r2-access-key",
			R2SecretKey:        "r2-secret-key",
			TurnstileURL:       "turnstile-url",
			TurnstileSiteKey:   "test-site-key",
			TurnstileSecretKey: "test-secret-key",
		},
	}

	app := fiber.New()

	app.Use(func(c fiber.Ctx) error {
		c.Locals("claims", &ComicsGaloreClaims{
			Name:           "Arthur Curry",
			UserID:         "user_77a1bc92",
			Email:          "arthur@atlantis.com",
			EmailVerified:  true,
			Picture:        "https://cdn.comics-galore.com/avatars/aquaman.png",
			NightMode:      true,
			CreatedAt:      time.Now().AddDate(0, -6, 0), // 6 months ago
			Role:           "user",
			IsBanned:       false,
			MembershipPlan: "pro_monthly",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "comics-galore-auth",
				Subject:   "user_77a1bc92",
				Audience:  jwt.ClaimStrings{"comics-galore-web"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		})
		return c.Next()
	})

	// 2. Register the middleware and a test handler
	app.Use(GetAppContext(env))

	app.Get("/", func(c fiber.Ctx) error {
		// Retrieve the context using the helper we created earlier
		ctx := c.Context()
		appCtx := GetAppContext2(ctx) // Assuming this is your retrieval helper

		// 3. Assertions
		if appCtx == nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		// Verify Title formatting from fullTitle()
		expectedTitle := "Comics-Galore - development"
		if appCtx.Title != expectedTitle {
			t.Errorf("expected title %s, got %s", expectedTitle, appCtx.Title)
		}

		// Verify Turnstile key
		if appCtx.TurnstileSiteKey != "test-site-key" {
			t.Errorf("expected site key %s, got %s", "test-site-key", appCtx.TurnstileSiteKey)
		}

		return c.SendString("ok")
	})

	// 4. Test execution
	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status OK, got %d", resp.StatusCode)
	}
}

func TestFullTitle(t *testing.T) {
	// 5. Explicitly test the helper function
	tests := []struct {
		title    string
		env      string
		expected string
	}{
		{"Comics Galore", "production", "Comics Galore"},
		{"Comics Galore", "prd", "Comics Galore"},
		{"Comics Galore", "prod", "Comics Galore"},
		{"Comics Galore", "development", "Comics-Galore - development"},
		{"Comics Galore", "staging", "Comics-Galore - staging"},
	}

	for _, tt := range tests {
		actual := fullTitle(tt.title, tt.env)
		if actual != tt.expected {
			t.Errorf("fullTitle(%s, %s) = %s; want %s", tt.title, tt.env, actual, tt.expected)
		}
	}
}
