package validator

import (
	"github.com/flectolab/flecto-manager/crypt"
	"github.com/go-playground/validator/v10"
)

const EncryptionKeyKey = "encryption_key"

// ValidateEncryptionKey checks that a configured key decodes to a usable key.
// Catching it here rather than at first use is what keeps a typo in the vault
// from surfacing as an unreadable secret hours later.
func ValidateEncryptionKey(fl validator.FieldLevel) bool {
	_, err := crypt.ParseKey(fl.Field().String())
	return err == nil
}
