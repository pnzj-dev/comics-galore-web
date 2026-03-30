package auth

import (
	"comics-galore-web/cmd/web/handlers/view"
	"comics-galore-web/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

func (h *handler) generateJWT(session *auth.UserSession) (string, error) {
	// Before creating claims
	expiresAt := session.Session.ExpiresAt
	if expiresAt.IsZero() || expiresAt.Before(time.Now()) {
		// Fallback to a default (e.g., 24 hours from now) if the session data is missing
		expiresAt = time.Now().Add(30 * 24 * time.Hour)
	}

	// 1. Define the claims based on the Better-Auth data
	_claims := view.ComicsGaloreClaims{
		UserID:         session.User.ID,
		Email:          session.User.Email,
		Name:           session.User.Name,
		Role:           session.User.Role,
		Picture:        session.User.Image,
		IsBanned:       session.User.Banned,
		CreatedAt:      session.User.CreatedAt,
		BanReason:      session.User.BanReason,
		EmailVerified:  session.User.EmailVerified,
		MembershipPlan: session.User.MembershipPlan,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "comics-galore",
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// 2. Create the token using HS256 (Secret Key)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, _claims)

	// 3. Sign it with your backend secret
	return token.SignedString([]byte(h.cfg.Get().JwtSecret))
}
