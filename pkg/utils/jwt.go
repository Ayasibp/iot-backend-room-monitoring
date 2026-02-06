package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	accessSecret  string
	refreshSecret string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
)

// InitJWT initializes JWT secrets and expiry times
func InitJWT(accessSec, refreshSec string, accessExp, refreshExp time.Duration) {
	accessSecret = accessSec
	refreshSecret = refreshSec
	accessExpiry = accessExp
	refreshExpiry = refreshExp
}

// Claims represents JWT custom claims
type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken generates a short-lived JWT access token
func GenerateAccessToken(userID uint, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(accessSecret))
}

// GenerateRefreshToken generates a cryptographically random refresh token
func GenerateRefreshToken() (string, error) {
	return uuid.New().String(), nil
}

// ValidateAccessToken validates and parses a JWT access token
func ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(accessSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// HashRefreshToken creates a SHA-256 hash of the refresh token for secure storage
func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GetRefreshTokenExpiry returns the refresh token expiry duration
func GetRefreshTokenExpiry() time.Duration {
	return refreshExpiry
}
