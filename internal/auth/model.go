package auth

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"net/url"
	"time"
)

/*type ComicsGaloreClaims struct {

	// Custom Identity Claims
	Name          string `json:"name"`
	UserID        string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"ver"`
	Picture       string `json:"img,omitempty"`

	// Authorization & Status Claims
	Role           string `json:"role"`
	IsBanned       bool   `json:"ban"`
	BanReason      string `json:"banr"`
	MembershipPlan string `json:"plan"`

	// Standard Claims
	jwt.RegisteredClaims
}*/

type UserSession struct {
	User    UserInfo    `json:"user"`
	Session SessionInfo `json:"session"` // Assuming Better-Auth returns a session object alongside user
}

type UserInfo struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	EmailVerified  bool       `json:"emailVerified"`
	Image          string     `json:"image"`
	Role           string     `json:"role"`
	NightMode      bool       `json:"nightMode,omitempty"`
	Banned         bool       `json:"banned"`
	BanReason      string     `json:"banReason"`
	BanExpires     *time.Time `json:"banExpires"` // Pointer for nullability
	MembershipPlan string     `json:"membershipPlan"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	jwt.RegisteredClaims
}

type SessionInfo struct {
	ExpiresAt time.Time `json:"expiresAt"`
	Token     string    `json:"token"`
}

func (u *UserInfo) AvatarUrl() string {
	if u.Image != "" {
		return u.Image
	}
	if u.Name != "" {
		return fmt.Sprintf("https://ui-avatars.com/api/?name=%s", url.QueryEscape(u.Name))
	}
	return "/assets/images/image-avatar-placeholder.svg"
}

func (u *UserInfo) AvatarAlt() string {
	return u.Email + "'s avatar"
}

func (u *UserInfo) GetFirstLetter() string {
	runes := []rune(u.Name)
	if len(runes) > 0 {
		return string(runes[0])
	}
	return ""
}
