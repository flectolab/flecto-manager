package crypt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptString(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := newTestCipher(t)

		payload, err := EncryptString(c, "hello")

		assert.NoError(t, err)
		assert.NotEqual(t, "hello", payload)
	})

	t.Run("failWithoutKey", func(t *testing.T) {
		_, err := EncryptString(NewUnavailable(), "hello")

		assert.ErrorIs(t, err, ErrNoKey)
	})
}

func TestDecryptString(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := newTestCipher(t)
		payload, err := EncryptString(c, "hello")
		assert.NoError(t, err)

		got, err := DecryptString(c, payload)

		assert.NoError(t, err)
		assert.Equal(t, "hello", got)
	})

	t.Run("failWithMalformedPayload", func(t *testing.T) {
		got, err := DecryptString(newTestCipher(t), "not-an-envelope")

		assert.ErrorIs(t, err, ErrMalformedPayload)
		assert.Equal(t, "", got)
	})
}

func TestNewUnavailable(t *testing.T) {
	c := NewUnavailable()

	_, errEncrypt := c.Encrypt([]byte("hello"))
	_, errDecrypt := c.Decrypt("v1.abcd1234.body")

	assert.ErrorIs(t, errEncrypt, ErrNoKey)
	assert.ErrorIs(t, errDecrypt, ErrNoKey)
}

func TestIsUnavailable(t *testing.T) {
	assert.True(t, IsUnavailable(NewUnavailable()))
	assert.False(t, IsUnavailable(newTestCipher(t)))
}

func TestNewFromKey(t *testing.T) {
	tests := []struct {
		name            string
		value           string
		wantErr         assert.ErrorAssertionFunc
		wantUnavailable bool
	}{
		{
			name:            "successWithoutValue",
			value:           "",
			wantErr:         assert.NoError,
			wantUnavailable: true,
		},
		{
			name:    "successWithKey",
			value:   EncodeKey(make([]byte, KeySize)),
			wantErr: assert.NoError,
		},
		{
			name:    "failWithInvalidKey",
			value:   "not-a-key",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFromKey(tt.value)

			tt.wantErr(t, err)
			if err != nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.wantUnavailable, IsUnavailable(got))
		})
	}
}

// newTestCipher builds a cipher on a fixed test key.
func newTestCipher(t *testing.T) Cipher {
	t.Helper()

	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}

	c, err := NewAESGCM(key)
	assert.NoError(t, err)
	return c
}
