package auth2

type AuthModalVM struct {
	Tab              string
	Errors           map[string]string
	Values           map[string]string
	TurnstileSiteKey string
}
