// Package crypt provides symmetric encryption for values that must be stored at
// rest, such as third-party credentials kept in the database.
//
// It knows nothing about what it protects: any feature needing a secret column
// takes a Cipher, or tags its field with the "encrypted" GORM serializer.
package crypt

import "errors"

var (
	// ErrNoKey is returned by every operation of the cipher built when no
	// encryption key is configured. Callers surface it as "feature unavailable"
	// rather than failing at boot: an instance with no key must still start and
	// serve everything that does not need encryption.
	ErrNoKey = errors.New("no encryption key configured")

	// ErrMalformedPayload is returned when a stored value does not have the
	// expected envelope, which means it was not produced by this package.
	ErrMalformedPayload = errors.New("malformed encrypted payload")

	// ErrUnknownKey is returned when a payload references a key id that is not in
	// the configured key set, typically after a key was rotated out too early.
	ErrUnknownKey = errors.New("unknown encryption key id")
)

// Cipher encrypts and decrypts values kept at rest.
//
// Encrypt returns a self-describing envelope: it carries the key id used, so a
// value stays readable after a new key becomes the active one.
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

// unavailable is the cipher used when no key is configured. It fails every call
// instead of silently storing plaintext.
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

// IsUnavailable reports whether c is the cipher returned by NewUnavailable, which
// is how a service tells "encryption is not configured" from "encryption failed".
func IsUnavailable(c Cipher) bool {
	_, ok := c.(unavailable)
	return ok
}

// NewFromKey builds a cipher from a configured key value. An empty value yields
// the unavailable cipher, so a deployment that needs no encrypted feature starts
// without any key material.
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
