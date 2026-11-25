// Package crypt provides symmetric encryption for values stored at rest, through
// a Cipher or the "encrypted" GORM serializer.
package crypt

import "errors"

var (
	// ErrNoKey is returned when no encryption key is configured; callers report the
	// feature as unavailable instead of failing at boot.
	ErrNoKey = errors.New("no encryption key configured")

	// ErrMalformedPayload is returned when a stored value is not a valid envelope.
	ErrMalformedPayload = errors.New("malformed encrypted payload")

	// ErrUnknownKey is returned when a payload references a key id not configured.
	ErrUnknownKey = errors.New("unknown encryption key id")
)

// Cipher encrypts and decrypts values kept at rest. Envelopes carry the key id
// used, so values stay readable after a key rotation.
type Cipher interface {
	Encrypt(plaintext []byte) (string, error)
	Decrypt(payload string) ([]byte, error)
}

// EncryptString is the string flavour of Cipher.Encrypt.
func EncryptString(c Cipher, plaintext string) (string, error) {
	return c.Encrypt([]byte(plaintext))
}

// DecryptString is the string flavour of Cipher.Decrypt.
func DecryptString(c Cipher, payload string) (string, error) {
	plaintext, err := c.Decrypt(payload)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// unavailable is the cipher used when no key is configured.
type unavailable struct{}

// NewUnavailable returns a Cipher that refuses every operation with ErrNoKey.
func NewUnavailable() Cipher {
	return unavailable{}
}

func (unavailable) Encrypt([]byte) (string, error) {
	return "", ErrNoKey
}

func (unavailable) Decrypt(string) ([]byte, error) {
	return nil, ErrNoKey
}

// IsUnavailable reports whether c is the cipher returned by NewUnavailable.
func IsUnavailable(c Cipher) bool {
	_, ok := c.(unavailable)
	return ok
}

// NewFromKey builds a cipher from a configured key; an empty value yields the
// unavailable cipher.
func NewFromKey(value string) (Cipher, error) {
	if value == "" {
		return NewUnavailable(), nil
	}

	key, err := ParseKey(value)
	if err != nil {
		return nil, err
	}
	return NewAESGCM(key)
}
