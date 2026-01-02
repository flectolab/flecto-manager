package cli

import (
	"errors"
	"fmt"

	"github.com/flectolab/flecto-manager/context"
	flectoValidator "github.com/flectolab/flecto-manager/validator"
	"github.com/go-playground/validator/v10"
)

func validateConfig(ctx *context.Context) error {
	validate := flectoValidator.New()
	err := validate.Struct(ctx.Config)
	if err != nil {

		var validationErrors validator.ValidationErrors
		switch {
		case errors.As(err, &validationErrors):
			for _, validationError := range validationErrors {
				ctx.Logger.Error(fmt.Sprintf("%v", validationError))
			}
			return errors.New("configuration file is not valid")
		default:
			return err
		}
	}
	return nil
}
