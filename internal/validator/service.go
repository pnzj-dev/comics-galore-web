package validator

import (
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/fr"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofiber/fiber/v3"
)

type Service interface {
	Validate(out any) error
}

type service struct {
	uni       *ut.UniversalTranslator
	trans     ut.Translator
	validator *validator.Validate
}

func NewService() Service {
	validate := validator.New()
	uni := ut.New(en.New(), fr.New())
	trans, _ := uni.GetTranslator("en")
	_ = entranslations.RegisterDefaultTranslations(validate, trans)
	return &service{
		uni:       ut.New(en.New(), fr.New()),
		trans:     trans,
		validator: validator.New(),
	}
}

func setupValidator(app *fiber.App) {
	//app.Config().StructValidator = &structValidator{validate: validate}
}

/*
if ve, ok := err.(validator.ValidationErrors); ok {
    for _, fe := range ve {
        field := fe.Field()
        msg := fe.Translate(trans) // "Email is a required field", etc.
        errorsMap[field] = msg
    }
}
*/

func (v *service) Validate(out any) error {
	return v.validator.Struct(out)
}

/*
// internal/config/validator.go
import (
    "github.com/go-playground/validator/v10"
    en_translations "github.com/go-playground/validator/v10/translations/en"
    "github.com/go-playground/universal-translator"
    "golang.org/x/text/language"
)

var validate *validator.Validate
var trans ut.Translator

func SetupValidator() {
    validate = validator.New()
    en := en.New()
    uni := ut.New(en, en)
    trans, _ = uni.GetTranslator("en")

    // Register default + custom precise messages
    _ = en_translations.RegisterDefaultTranslations(validate, trans)

    // Custom overrides for even better messages
    _ = validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
        return ut.Add("required", "{0} is required", true)
    }, func(ut ut.Translator, fe validator.FieldError) string {
        t, _ := ut.T("required", fe.Field())
        return t
    })

    _ = validate.RegisterTranslation("email", trans, func(ut ut.Translator) error {
        return ut.Add("email", "Please enter a valid email address", true)
    }, func(ut ut.Translator, fe validator.FieldError) string {
        t, _ := ut.T("email")
        return t
    })

    _ = validate.RegisterTranslation("min", trans, func(ut ut.Translator) error {
        return ut.Add("min", "{0} must be at least {1} characters", true)
    }, func(ut ut.Translator, fe validator.FieldError) string {
        t, _ := ut.T("min", fe.Field(), fe.Param())
        return t
    })

    _ = validate.RegisterTranslation("eqfield", trans, func(ut ut.Translator) error {
        return ut.Add("eqfield", "Passwords do not match", true)
    }, func(ut ut.Translator, fe validator.FieldError) string {
        t, _ := ut.T("eqfield")
        return t
    })
}
*/
