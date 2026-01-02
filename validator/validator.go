package validator

import (
	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/go-playground/validator/v10"
)

func New(options ...validator.Option) *validator.Validate {
	validate := validator.New()
	_ = validate.RegisterValidation(CodeKey, ValidateCode)
	validate.RegisterStructValidation(ValidateRedirect, commonTypes.Redirect{})
	validate.RegisterStructValidation(ValidatePage, commonTypes.Page{})
	return validate
}
