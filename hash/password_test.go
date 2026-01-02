package hash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestPassword(t *testing.T) {
	t.Run("success hashes password", func(t *testing.T) {
		password := "mySecretPassword123"

		hashed, err := Password(password)

		assert.NoError(t, err)
		assert.NotNil(t, hashed)
		assert.NotEmpty(t, hashed)
		assert.NotEqual(t, password, string(hashed))
	})

	t.Run("hashed password can be verified", func(t *testing.T) {
		password := "testPassword456"

		hashed, err := Password(password)
		assert.NoError(t, err)

		err = bcrypt.CompareHashAndPassword(hashed, []byte(password))
		assert.NoError(t, err)
	})

	t.Run("wrong password fails verification", func(t *testing.T) {
		password := "correctPassword"
		wrongPassword := "wrongPassword"

		hashed, err := Password(password)
		assert.NoError(t, err)

		err = bcrypt.CompareHashAndPassword(hashed, []byte(wrongPassword))
		assert.Error(t, err)
	})

	t.Run("same password produces different hashes (salted)", func(t *testing.T) {
		password := "samePassword"

		hash1, err := Password(password)
		assert.NoError(t, err)

		hash2, err := Password(password)
		assert.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("empty password", func(t *testing.T) {
		hashed, err := Password("")

		assert.NoError(t, err)
		assert.NotEmpty(t, hashed)
	})

	t.Run("uses correct bcrypt cost", func(t *testing.T) {
		password := "testCost"

		hashed, err := Password(password)
		assert.NoError(t, err)

		cost, err := bcrypt.Cost(hashed)
		assert.NoError(t, err)
		assert.Equal(t, BcryptCost, cost)
	})
}

func TestCheckPassword(t *testing.T) {
	t.Run("success with correct password", func(t *testing.T) {
		password := "mySecretPassword123"
		hashed, err := Password(password)
		assert.NoError(t, err)

		err = CheckPassword(string(hashed), password)
		assert.NoError(t, err)
	})

	t.Run("error with wrong password", func(t *testing.T) {
		password := "correctPassword"
		wrongPassword := "wrongPassword"
		hashed, err := Password(password)
		assert.NoError(t, err)

		err = CheckPassword(string(hashed), wrongPassword)
		assert.Error(t, err)
		assert.Equal(t, bcrypt.ErrMismatchedHashAndPassword, err)
	})

	t.Run("error with invalid hash", func(t *testing.T) {
		err := CheckPassword("invalidhash", "password")
		assert.Error(t, err)
	})

	t.Run("error with empty hash", func(t *testing.T) {
		err := CheckPassword("", "password")
		assert.Error(t, err)
	})

	t.Run("success with empty password if hashed empty", func(t *testing.T) {
		hashed, err := Password("")
		assert.NoError(t, err)

		err = CheckPassword(string(hashed), "")
		assert.NoError(t, err)
	})
}