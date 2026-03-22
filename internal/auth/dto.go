package auth

type LoginInput struct {
	Email    string `json:"email" form:"email" validate:"required,email"`
	Password string `json:"password" form:"password" validate:"required,min=8"`
	// cf-turnstile-response is handled separately (not in body JSON usually)
}

type SignupInput struct {
	Name            string `json:"name" form:"name" validate:"omitempty,max=100"`
	Email           string `json:"email" form:"email" validate:"required,email"`
	Password        string `json:"password" form:"password" validate:"required,min=10"`
	ConfirmPassword string `json:"confirm_password" form:"confirm_password" validate:"required,eqfield=Password"`
}

type ForgotInput struct {
	Email string `json:"email" form:"email" validate:"required,email"`
}
