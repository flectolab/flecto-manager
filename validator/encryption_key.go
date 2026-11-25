package validator

import (
	"github.com/flectolab/flecto-manager/crypt"
	"github.com/go-playground/validator/v10"
)

const EncryptionKeyKey = "encryption_key"

// ValidateEncryptionKey checks at boot that a configured key decodes to a usable key.
func ValidateEncryptionKey(fl validator.FieldLevel) bool {
	_, err := crypt.ParseKey(fl.Field().String())
	return err == nil
}
