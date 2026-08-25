package validator

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/flectolab/flecto-manager/crypt"
	"github.com/stretchr/testify/assert"
)

func TestValidateEncryptionKey(t *testing.T) {
	key := make([]byte, crypt.KeySize)

	type payload struct {
		Key string `validate:"omitempty,encryption_key"`
	}

	tests := []struct {
		name    string
		key     string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "successWithBase64",
			key:     base64.StdEncoding.EncodeToString(key),
			wantErr: assert.NoError,
		},
		{
			name:    "successWithHexPrefix",
			key:     "hex:" + hex.EncodeToString(key),
			wantErr: assert.NoError,
		},
		{
			name:    "successWithEmptyKey",
			key:     "",
			wantErr: assert.NoError,
		},
		{
			name:    "failWithGarbage",
			key:     "not-a-key",
			wantErr: assert.Error,
		},
		{
			name:    "failWithWrongLength",
			key:     base64.StdEncoding.EncodeToString([]byte("too-short")),
			wantErr: assert.Error,
		},
	}

	validate := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(payload{Key: tt.key})

			tt.wantErr(t, err)
		})
	}
}
