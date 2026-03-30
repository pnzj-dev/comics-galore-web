package config

import (
	"github.com/MicahParks/keyfunc/v3"
	"github.com/gofiber/storage/s3/v2"
	"time"
)

type Env struct {
	Port              int    `env:"PORT" envDefault:"8080"`
	Version           string `env:"VERSION" envDefault:"0.0.1"`
	AppEnv            string `env:"APP_ENV" envDefault:"local"`
	DatabaseDSN       string `env:"DATABASE_DSN"`
	MaxCommentNesting int    `env:"MAX_COMMENT_NESTING" envDefault:"5"`
	BetterAuth        string `env:"BETTER_AUTH" envDefault:"https://auth.comics-galore.com/api/auth/"`
	JwtSecret         string `env:"JWT_SECRET" envDefault:"7f8d6263836b47c6981882d2d38510842e2b3e45f9e9a4d293816b3281907421"`
	BetterAuthSecret  string `env:"BETTER_AUTH_SECRET"`
	JwksUrl           string `env:"JWKS_URL"`
	JwksFunc          keyfunc.Keyfunc
	SessionKey        string   `env:"SESSION_KEY" envDefault:"comics-galore.session_token"`
	AllowedRoles      []string `env:"ALLOWED_ROLES" envDefault:"admin,editor,user"`

	// AWS Configuration
	AWS struct {
		Region          string `env:"AWS_REGION"`
		S3Bucket        string `env:"AWS_S3_BUCKET"`
		S3Endpoint      string `env:"AWS_S3_ENDPOINT"`
		AccessKeyID     string `env:"AWS_ACCESS_KEY_ID"`
		SecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY"`
	}

	// Cloudflare (Images & R2)
	Cloudflare struct {
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
	}

	// SendGrid
	SendGrid struct {
		APIKey   string `env:"SENDGRID_APIKEY"`
		Endpoint string `env:"SENDGRID_ENDPOINT"`
		Username string `env:"SENDGRID_USERNAME"`
		SMTPHost string `env:"SENDGRID_SMTP_HOST"`
		Key      string `env:"SENDGRID_KEY"` // Note: often same as APIKey
	}

	// X402 Payment/Network
	X402 struct {
		EVMNetwork      string `env:"X402_EVM_NETWORK"`
		FacilitatorURL  string `env:"X402_FACILITATOR_URL"`
		EVMPayeeAddress string `env:"X402_EVM_PAYEE_ADDRESS"`
	}

	// NowPayments
	NowPayments struct {
		APIKey    string `env:"NOWPAYMENTS_APIKEY"`
		Endpoint  string `env:"NOWPAYMENTS_ENDPOINT"`
		IPNSecret string `env:"NOWPAYMENTS_IPN_SECRET"`
	}
}

func (e *Env) S3Config() *s3.Config {
	return &s3.Config{
		Bucket:         e.AWS.S3Bucket,
		Endpoint:       e.AWS.S3Endpoint,
		Region:         e.AWS.Region,
		RequestTimeout: 30 * time.Second,
		//RequestTimeout: 5 * time.Minute,
		Reset: false,
		Credentials: s3.Credentials{
			AccessKey:       e.AWS.AccessKeyID,
			SecretAccessKey: e.AWS.SecretAccessKey,
		},
		MaxAttempts: 3,
	}
}
