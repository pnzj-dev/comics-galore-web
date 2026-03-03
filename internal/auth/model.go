package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Session struct {
	ExpiresAt      time.Time `json:"expiresAt"`
	Token          string    `json:"token"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	IpAddress      string    `json:"ipAddress"`
	UserAgent      string    `json:"userAgent"`
	UserId         string    `json:"userId"`
	ImpersonatedBy *string   `json:"impersonatedBy"`
	Id             string    `json:"id"`
}

type User struct {
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	EmailVerified  bool      `json:"emailVerified"`
	Image          *string   `json:"image"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Role           string    `json:"role"`
	Banned         bool      `json:"banned"`
	BanReason      *string   `json:"banReason"`
	BanExpires     *string   `json:"banExpires"`
	MembershipPlan string    `json:"membershipPlan"`
	Id             string    `json:"id"`
}

type GetSession struct {
	Session Session `json:"session"`
	User    User    `json:"user"`
}

type Claims struct {
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	EmailVerified  bool      `json:"emailVerified"`
	Image          *string   `json:"image"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Role           string    `json:"role"`
	Banned         bool      `json:"banned"`
	BanReason      *string   `json:"banReason"`
	BanExpires     *string   `json:"banExpires"`
	MembershipPlan string    `json:"membershipPlan"`
	Id             string    `json:"id"`
	jwt.RegisteredClaims
}

func (c *Claims) AvatarUrl() string {
	if c.Image != nil {
		return *c.Image
	}
	return ""
}

func (c *Claims) AvatarAlt() string {
	return c.Email + "'s avatar"
}
