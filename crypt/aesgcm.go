package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// KeySize is the key length required by AES-256.
const KeySize = 32

// envelopeVersion prefixes every payload. It is the hook for changing the scheme
// later without having to guess how an existing value was produced.
const envelopeVersion = "v1"

// keyIDLength is the number of hex characters of the key fingerprint kept in the
// envelope. It identifies a key among the configured ones, so it only has to be
// collision-free within a handful of keys, and it is a fingerprint rather than an
// operator-chosen name so that rotation needs no extra configuration.
const keyIDLength = 8

// aesGCM encrypts with AES-256-GCM. Keys are indexed by fingerprint: the active
// one encrypts, any of them decrypts, which is what makes a key rotation a matter
// of adding the new key and re-writing values in the background.
type aesGCM struct {
	active  string
	aeads   map[string]cipher.AEAD
	randSrc func([]byte) (int, error)
}

// NewAESGCM builds a Cipher from an active key and any number of retired keys
// still needed to read old values. Every key must be KeySize bytes long.
func NewAESGCM(activeKey []byte, previousKeys ...[]byte) (Cipher, error) {
	c := &aesGCM{
		aeads:   make(map[string]cipher.AEAD, 1+len(previousKeys)),
		randSrc: rand.Read,
	}

	activeID, err := c.addKey(activeKey)
	if err != nil {
		return nil, err
	}
	c.active = activeID

	for _, key := range previousKeys {
		if _, err = c.addKey(key); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *aesGCM) addKey(key []byte) (string, error) {
	if len(key) != KeySize {
		return "", fmt.Errorf("encryption key must be %d bytes, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	id := KeyID(key)
	c.aeads[id] = aead
	return id, nil
}

// Encrypt seals the plaintext with the active key and returns the envelope
// "v1.<keyID>.<base64(nonce||ciphertext)>".
func (c *aesGCM) Encrypt(plaintext []byte) (string, error) {
	aead := c.aeads[c.active]

	nonce := make([]byte, aead.NonceSize())
	if _, err := c.randSrc(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// The nonce is prepended to the ciphertext rather than kept in its own envelope
	// field: it is not a secret, and one less field is one less thing to parse.
	sealed := aead.Seal(nonce, nonce, plaintext, nil)

	return strings.Join([]string{
		envelopeVersion,
		c.active,
		base64.RawStdEncoding.EncodeToString(sealed),
	}, "."), nil
}

func (c *aesGCM) Decrypt(payload string) ([]byte, error) {
	version, keyID, body, err := splitEnvelope(payload)
	if err != nil {
		return nil, err
	}
	if version != envelopeVersion {
		return nil, fmt.Errorf("%w: unsupported version %q", ErrMalformedPayload, version)
	}

	aead, ok := c.aeads[keyID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownKey, keyID)
	}

	sealed, err := base64.RawStdEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
	}
	if len(sealed) < aead.NonceSize() {
		return nil, fmt.Errorf("%w: payload shorter than nonce", ErrMalformedPayload)
	}

	nonce, ciphertext := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// The underlying error only ever says "message authentication failed", which
		// covers a wrong key as well as a tampered value.
		return nil, fmt.Errorf("failed to decrypt payload: %w", err)
	}
	return plaintext, nil
}

// splitEnvelope cuts "version.keyID.body" apart. The body is base64 and never
// contains a dot, so splitting on the first two is unambiguous.
func splitEnvelope(payload string) (version, keyID, body string, err error) {
	parts := strings.SplitN(payload, ".", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", ErrMalformedPayload
	}
	return parts[0], parts[1], parts[2], nil
}

// KeyID is the fingerprint identifying a key inside an envelope. It is a
// truncated hash: knowing it tells nothing about the key itself.
func KeyID(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:])[:keyIDLength]
}

// ParseKey reads a key from its configured form. It accepts an explicit
// "base64:" or "hex:" prefix, and falls back to base64 so that the common case
// stays a bare value.
func ParseKey(value string) ([]byte, error) {
	var (
		key []byte
		err error
	)

	switch {
	case strings.HasPrefix(value, "hex:"):
		key, err = hex.DecodeString(strings.TrimPrefix(value, "hex:"))
	case strings.HasPrefix(value, "base64:"):
		key, err = decodeBase64(strings.TrimPrefix(value, "base64:"))
	default:
		key, err = decodeBase64(value)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key: %w", err)
	}

	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid encryption key: must decode to %d bytes, got %d", KeySize, len(key))
	}
	return key, nil
}

// decodeBase64 accepts both the padded and unpadded forms, which is the kind of
// difference an operator should not have to care about.
func decodeBase64(value string) ([]byte, error) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(value)
}

// GenerateKey draws a new key of the required size. It is what the key
// generation command hands to an operator, never called at runtime: a key the
// application makes up for itself would be lost at the next restart, taking every
// encrypted value with it.
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}
	return key, nil
}

// EncodeKey renders a key in the form ParseKey reads back.
func EncodeKey(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}
