package validator

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestValidateCode(t *testing.T) {
	type args struct {
		String string `validate:"code"`
	}
	validate := validator.New()
	_ = validate.RegisterValidation(CodeKey, ValidateCode)

	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "successAlpha",
			args:    args{String: "foo"},
			wantErr: assert.NoError,
		},
		{
			name:    "successAlphaNumeric",
			args:    args{String: "foo1"},
			wantErr: assert.NoError,
		},
		{
			name:    "successAlphaNumericUnderscore",
			args:    args{String: "foo_1"},
			wantErr: assert.NoError,
		},
		{
			name:    "successAlphaNumericHyphen",
			args:    args{String: "foo-1"},
			wantErr: assert.NoError,
		},
		{
			name:    "failWithSpace",
			args:    args{String: "foo 1"},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.args)
			tt.wantErr(t, err, "Code is not valid")
		})
	}
}
