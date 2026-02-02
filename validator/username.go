package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

const (
	usernameRegexString = `^[a-zA-Z0-9_-]+$`
	emailRegexString    = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	UsernameKey         = "username"
)

var (
	usernameRegex = regexp.MustCompile(usernameRegexString)
	emailRegex    = regexp.MustCompile(emailRegexString)
)

// ValidateUsername accepts either a code format (alphanumeric with _ and -)
// or a valid email address format
func ValidateUsername(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return usernameRegex.MatchString(value) || emailRegex.MatchString(value)
}