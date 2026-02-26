package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

type User struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Email          string             `json:"email"`
	EmailVerified  bool               `json:"email_verified"`
	Image          *string            `json:"image"`
	Role           interface{}        `json:"role"`
	MembershipPlan *string            `json:"membership_plan"`
	Banned         *bool              `json:"banned"`
	BanReason      *string            `json:"ban_reason"`
	BanExpires     pgtype.Timestamptz `json:"ban_expires"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type BetterAuthClaims struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`

	// Standard Better-Auth Fields
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Image   string `json:"image"`

	// Your Custom Fields (Better-Auth maps these if configured)
	Role           string `json:"role"`
	Banned         bool   `json:"banned"`
	MembershipPlan string `json:"membership_plan"`

	jwt.RegisteredClaims
}
