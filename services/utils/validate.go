package utils

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/nbittich/wtm/types"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
	Validate.RegisterValidation("password", validatePassword)
	Validate.RegisterValidation("startswithalpha", validateStartsWithAlpha)
}

func ValidateStruct(s interface{}) error {
	err := Validate.Struct(s)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			panic(err) // should never happen
		}

		validationErrors := err.(validator.ValidationErrors)
		errors := make(types.InvalidMessage, len(validationErrors)+1)
		for _, err := range validationErrors {
			errors[strings.ToLower(err.Field()[0:1])+err.Field()[1:]] = err.Tag()
		}
		return types.InvalidFormError{Messages: errors}
	}
	return nil
}

func validateStartsWithAlpha(fl validator.FieldLevel) bool {
	str := fl.Field().String()

	if len(str) == 0 {
		return false // Invalid if the string is empty
	}

	return unicode.IsLetter(rune(str[0]))
}

func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Check for at least one uppercase letter
	if !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return false
	}

	// Check for at least one lowercase letter
	if !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		return false
	}

	// Check for at least one special character
	specialChars := regexp.MustCompile(`[^a-zA-Z0-9]`)

	return specialChars.MatchString(password)
}
