package hash

import "golang.org/x/crypto/bcrypt"

const (
	BcryptCost = 12
)

func Password(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
}

// CheckPassword compares a plaintext password with a hashed password
func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
