package crypt

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAESGCM(t *testing.T) {
	tests := []struct {
		name     string
		active   []byte
		previous [][]byte
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:    "success",
			active:  testKey(0x01),
			wantErr: assert.NoError,
		},
		{
			name:     "successWithPreviousKeys",
			active:   testKey(0x01),
			previous: [][]byte{testKey(0x02), testKey(0x03)},
			wantErr:  assert.NoError,
		},
		{
			name:    "failWithShortActiveKey",
			active:  []byte("too-short"),
			wantErr: assert.Error,
		},
		{
			name:     "failWithShortPreviousKey",
			active:   testKey(0x01),
			previous: [][]byte{[]byte("too-short")},
			wantErr:  assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAESGCM(tt.active, tt.previous...)

			tt.wantErr(t, err)
			if err == nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestAesGCM_Encrypt(t *testing.T) {
	t.Run("successProducesVersionedEnvelope", func(t *testing.T) {
		key := testKey(0x01)
		c, err := NewAESGCM(key)
		assert.NoError(t, err)

		payload, err := c.Encrypt([]byte("s3cret"))

		assert.NoError(t, err)
		parts := strings.Split(payload, ".")
		assert.Len(t, parts, 3)
		assert.Equal(t, envelopeVersion, parts[0])
		assert.Equal(t, KeyID(key), parts[1])
		assert.NotContains(t, payload, "s3cret")
	})

	t.Run("successProducesDistinctPayloadsForSamePlaintext", func(t *testing.T) {
		c, err := NewAESGCM(testKey(0x01))
		assert.NoError(t, err)

		first, err := c.Encrypt([]byte("s3cret"))
		assert.NoError(t, err)
		second, err := c.Encrypt([]byte("s3cret"))
		assert.NoError(t, err)

		// A fresh nonce per call is what keeps two identical secrets from being
		// recognisable as identical in a database dump.
		assert.NotEqual(t, first, second)
	})

	t.Run("successWithEmptyPlaintext", func(t *testing.T) {
		c, err := NewAESGCM(testKey(0x01))
		assert.NoError(t, err)

		payload, err := c.Encrypt(nil)
		assert.NoError(t, err)

		got, err := c.Decrypt(payload)
		assert.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("failWhenNonceCannotBeGenerated", func(t *testing.T) {
		c, err := NewAESGCM(testKey(0x01))
		assert.NoError(t, err)
		c.(*aesGCM).randSrc = func([]byte) (int, error) {
			return 0, errors.New("no entropy")
		}

		_, err = c.Encrypt([]byte("s3cret"))

		assert.ErrorContains(t, err, "failed to generate nonce")
	})
}

func TestAesGCM_Decrypt(t *testing.T) {
	activeKey := testKey(0x01)
	retiredKey := testKey(0x02)

	t.Run("successRoundTrip", func(t *testing.T) {
		c, err := NewAESGCM(activeKey)
		assert.NoError(t, err)
		payload, err := c.Encrypt([]byte("s3cret"))
		assert.NoError(t, err)

		got, err := c.Decrypt(payload)

		assert.NoError(t, err)
		assert.Equal(t, []byte("s3cret"), got)
	})

	t.Run("successWithRetiredKey", func(t *testing.T) {
		old, err := NewAESGCM(retiredKey)
		assert.NoError(t, err)
		payload, err := old.Encrypt([]byte("s3cret"))
		assert.NoError(t, err)

		// The retired key is kept only for reading, the active one is now different.
		rotated, err := NewAESGCM(activeKey, retiredKey)
		assert.NoError(t, err)

		got, err := rotated.Decrypt(payload)

		assert.NoError(t, err)
		assert.Equal(t, []byte("s3cret"), got)
	})

	t.Run("failWithUnknownKey", func(t *testing.T) {
		old, err := NewAESGCM(retiredKey)
		assert.NoError(t, err)
		payload, err := old.Encrypt([]byte("s3cret"))
		assert.NoError(t, err)

		c, err := NewAESGCM(activeKey)
		assert.NoError(t, err)

		_, err = c.Decrypt(payload)

		assert.ErrorIs(t, err, ErrUnknownKey)
	})

	t.Run("failWithTamperedPayload", func(t *testing.T) {
		c, err := NewAESGCM(activeKey)
		assert.NoError(t, err)
		payload, err := c.Encrypt([]byte("s3cret"))
		assert.NoError(t, err)

		parts := strings.Split(payload, ".")
		sealed, err := base64.RawStdEncoding.DecodeString(parts[2])
		assert.NoError(t, err)
		sealed[len(sealed)-1] ^= 0xff
		parts[2] = base64.RawStdEncoding.EncodeToString(sealed)

		_, err = c.Decrypt(strings.Join(parts, "."))

		assert.ErrorContains(t, err, "failed to decrypt payload")
	})

	tests := []struct {
		name    string
		payload string
	}{
		{name: "failWithEmptyPayload", payload: ""},
		{name: "failWithoutSeparator", payload: "v1"},
		{name: "failWithEmptyBody", payload: "v1.abcd1234."},
		{name: "failWithUnsupportedVersion", payload: "v2.abcd1234.body"},
		{name: "failWithNonBase64Body", payload: "v1." + KeyID(activeKey) + ".not-base64!"},
		{name: "failWithBodyShorterThanNonce", payload: "v1." + KeyID(activeKey) + "." + base64.RawStdEncoding.EncodeToString([]byte("short"))},
	}

	c, err := NewAESGCM(activeKey)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errDecrypt := c.Decrypt(tt.payload)

			assert.Error(t, errDecrypt)
		})
	}
}

func TestKeyID(t *testing.T) {
	t.Run("successIsStableAndShort", func(t *testing.T) {
		key := testKey(0x01)

		assert.Equal(t, KeyID(key), KeyID(key))
		assert.Len(t, KeyID(key), keyIDLength)
	})

	t.Run("successDiffersBetweenKeys", func(t *testing.T) {
		assert.NotEqual(t, KeyID(testKey(0x01)), KeyID(testKey(0x02)))
	})
}

func TestParseKey(t *testing.T) {
	key := testKey(0x01)

	tests := []struct {
		name    string
		value   string
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "successWithBareBase64",
			value:   base64.StdEncoding.EncodeToString(key),
			want:    key,
			wantErr: assert.NoError,
		},
		{
			name:    "successWithUnpaddedBase64",
			value:   base64.RawStdEncoding.EncodeToString(key),
			want:    key,
			wantErr: assert.NoError,
		},
		{
			name:    "successWithBase64Prefix",
			value:   "base64:" + base64.StdEncoding.EncodeToString(key),
			want:    key,
			wantErr: assert.NoError,
		},
		{
			name:    "successWithHexPrefix",
			value:   "hex:" + hex.EncodeToString(key),
			want:    key,
			wantErr: assert.NoError,
		},
		{
			name:    "failWithEmptyValue",
			value:   "",
			wantErr: assert.Error,
		},
		{
			name:    "failWithInvalidBase64",
			value:   "not base64 at all!",
			wantErr: assert.Error,
		},
		{
			name:    "failWithInvalidHex",
			value:   "hex:zzzz",
			wantErr: assert.Error,
		},
		{
			name:    "failWithWrongLength",
			value:   base64.StdEncoding.EncodeToString([]byte("too-short")),
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKey(tt.value)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// testKey builds a deterministic key of the required size, filled with a single
// byte so two calls with different fillers give two different keys.
func testKey(filler byte) []byte {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = filler
	}
	return key
}

func TestGenerateKey(t *testing.T) {
	t.Run("successProducesUsableKey", func(t *testing.T) {
		key, err := GenerateKey()

		assert.NoError(t, err)
		assert.Len(t, key, KeySize)

		_, err = NewAESGCM(key)
		assert.NoError(t, err)
	})

	t.Run("successProducesDistinctKeys", func(t *testing.T) {
		first, err := GenerateKey()
		assert.NoError(t, err)
		second, err := GenerateKey()
		assert.NoError(t, err)

		assert.NotEqual(t, first, second)
	})
}

func TestEncodeKey(t *testing.T) {
	t.Run("successRoundTripsThroughParseKey", func(t *testing.T) {
		key, err := GenerateKey()
		assert.NoError(t, err)

		got, err := ParseKey(EncodeKey(key))

		assert.NoError(t, err)
		assert.Equal(t, key, got)
	})
}
