package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

const (
	codeRegexString = `^[a-zA-Z0-9_-]+$`
	CodeKey         = "code"
)

var codeRegex = regexp.MustCompile(codeRegexString)

func ValidateCode(fl validator.FieldLevel) bool {
	return codeRegex.MatchString(fl.Field().String())
}
