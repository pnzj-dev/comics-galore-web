package view

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/golang-jwt/jwt/v5"
	"net/url"
	"time"
)

type ComicsGaloreClaims struct {

	// Custom Identity Claims
	Name          string    `json:"name"`
	UserID        string    `json:"sub"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"ver"`
	Picture       string    `json:"img,omitempty"`
	NightMode     bool      `json:"nightMode,omitempty"`
	CreatedAt     time.Time `json:"created"`

	// Authorization & Status Claims
	Role           string `json:"role"`
	IsBanned       bool   `json:"ban"`
	BanReason      string `json:"banr"`
	MembershipPlan string `json:"plan"`

	// Standard Claims
	jwt.RegisteredClaims
}

func GetClaims(c fiber.Ctx) *ComicsGaloreClaims {
	val := c.Locals("claims")

	// 1. Check if the key even exists
	if val == nil {
		log.Info("GetClaims: Key 'claims' does not exist in Locals")
		return nil
	}

	// 2. Debug: What is actually inside 'val'?
	log.Infof("GetClaims: Type of val is %T", val)

	// 3. Perform assertion
	claims, ok := val.(*ComicsGaloreClaims)

	// 4. Check 'ok' BEFORE logging/using userInfo
	if !ok || claims == nil {
		log.Warn("GetClaims: Assertion failed or pointer is nil")
		return nil
	}

	// Now it is safe to pretty print
	prettyJSON, _ := json.MarshalIndent(claims, "", "  ")
	log.Infof("GetClaims: Success => \n%s", string(prettyJSON))

	return claims
}

func (u *ComicsGaloreClaims) AvatarUrl() string {
	if u.Picture != "" {
		return u.Picture
	}
	if u.Name != "" {
		return fmt.Sprintf("https://ui-avatars.com/api/?name=%s", url.QueryEscape(u.Name))
	}
	return "/assets/images/image-avatar-placeholder.svg"
}

func (u *ComicsGaloreClaims) AvatarAlt() string {
	return u.Email + "'s avatar"
}

func (u *ComicsGaloreClaims) GetFirstLetter() string {
	runes := []rune(u.Name)
	if len(runes) > 0 {
		return string(runes[0])
	}
	return ""
}
