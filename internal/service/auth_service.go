package service

import (
	"errors"
	"fmt"
	"time"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/repository"
	"iot-backend-room-monitoring/pkg/utils"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	auditRepo *repository.AuditRepository
}

func NewAuthService(userRepo *repository.UserRepository, auditRepo *repository.AuditRepository) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		auditRepo: auditRepo,
	}
}

// LoginResponse represents the response structure for login
type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(username, password string) (*LoginResponse, error) {
	// Find user by username
	user, err := s.userRepo.FindUserByUsername(username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Compare password
	if !utils.ComparePassword(user.PasswordHash, password) {
		return nil, errors.New("invalid credentials")
	}

	// Generate access token
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Hash and store refresh token
	tokenHash := utils.HashRefreshToken(refreshToken)
	refreshTokenModel := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(utils.GetRefreshTokenExpiry()),
	}

	if err := s.userRepo.CreateRefreshToken(refreshTokenModel); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Log login action
	userIDPtr := &user.ID
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "user_login", fmt.Sprintf("User %s logged in", username))

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	}, nil
}

// RefreshAccessToken generates a new access token from a refresh token
func (s *AuthService) RefreshAccessToken(refreshToken string) (string, error) {
	// Hash the refresh token
	tokenHash := utils.HashRefreshToken(refreshToken)

	// Find refresh token in database
	token, err := s.userRepo.FindRefreshTokenByHash(tokenHash)
	if err != nil {
		return "", errors.New("invalid or revoked refresh token")
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		return "", errors.New("refresh token expired")
	}

	// Generate new access token
	accessToken, err := utils.GenerateAccessToken(token.User.ID, token.User.Role)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	return accessToken, nil
}

// Logout revokes a refresh token
func (s *AuthService) Logout(refreshToken string) error {
	// Hash the refresh token
	tokenHash := utils.HashRefreshToken(refreshToken)

	// Revoke the token
	if err := s.userRepo.RevokeRefreshTokenByHash(tokenHash); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// Register creates a new user account
func (s *AuthService) Register(username, password, role string) (*LoginResponse, error) {
	// Check if username already exists
	existingUser, err := s.userRepo.FindUserByUsername(username)
	if err == nil && existingUser != nil {
		return nil, errors.New("username already exists")
	}

	// Hash the password
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
	}

	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate access token
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Hash and store refresh token
	tokenHash := utils.HashRefreshToken(refreshToken)
	refreshTokenModel := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(utils.GetRefreshTokenExpiry()),
	}

	if err := s.userRepo.CreateRefreshToken(refreshTokenModel); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Log registration action
	userIDPtr := &user.ID
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "user_registration", fmt.Sprintf("User %s registered", username))

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	}, nil
}
