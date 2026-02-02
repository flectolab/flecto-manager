package validator

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestValidateUsername(t *testing.T) {
	validate := validator.New()
	_ = validate.RegisterValidation(UsernameKey, ValidateUsername)

	type testStruct struct {
		Username string `validate:"username"`
	}

	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		// Code format (alphanumeric with _ and -)
		{name: "valid code lowercase", value: "admin", isValid: true},
		{name: "valid code uppercase", value: "ADMIN", isValid: true},
		{name: "valid code mixed case", value: "Admin", isValid: true},
		{name: "valid code with numbers", value: "admin123", isValid: true},
		{name: "valid code with underscore", value: "admin_user", isValid: true},
		{name: "valid code with hyphen", value: "admin-user", isValid: true},
		{name: "valid code mixed", value: "Admin_User-123", isValid: true},

		// Email format
		{name: "valid email simple", value: "user@example.com", isValid: true},
		{name: "valid email with subdomain", value: "user@mail.example.com", isValid: true},
		{name: "valid email with plus", value: "user+tag@example.com", isValid: true},
		{name: "valid email with dots", value: "first.last@example.com", isValid: true},
		{name: "valid email with numbers", value: "user123@example.com", isValid: true},
		{name: "valid email with hyphen domain", value: "user@my-domain.com", isValid: true},

		// Invalid values
		{name: "invalid empty", value: "", isValid: false},
		{name: "invalid with space", value: "admin user", isValid: false},
		{name: "invalid with special char", value: "admin@local", isValid: false},
		{name: "invalid email no domain", value: "user@", isValid: false},
		{name: "invalid email no tld", value: "user@example", isValid: false},
		{name: "invalid email double at", value: "user@@example.com", isValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testStruct{Username: tt.value}
			err := validate.Struct(s)
			if tt.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}