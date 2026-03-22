package header

import "comics-galore-web/internal/auth"

type UserProps struct {
	IsLoggedIn bool
	Username   string
	AvatarURL  string
}

func NewUserProps(claims *auth.Claims) *UserProps {
	return &UserProps{
		IsLoggedIn: true,
		Username:   claims.Name,
		AvatarURL:  claims.AvatarUrl(),
	}
}

func (u *UserProps) GetFirstLetter() string {
	runes := []rune(u.Username)
	if len(runes) > 0 {
		return string(runes[0])
	}
	return ""
}

type NavItem struct {
	Label  string
	Href   string
	Active bool
}
