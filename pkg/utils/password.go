package utils

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// HashPassword generates a bcrypt hash from a plain text password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}

// ComparePassword compares a bcrypt hashed password with plain text password
func ComparePassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
