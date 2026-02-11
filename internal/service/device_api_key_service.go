package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type DeviceAPIKeyService struct {
	apiKeyRepo *repository.DeviceAPIKeyRepository
	roomRepo   *repository.RoomRepository
	auditRepo  *repository.AuditRepository
}

func NewDeviceAPIKeyService(
	apiKeyRepo *repository.DeviceAPIKeyRepository,
	roomRepo *repository.RoomRepository,
	auditRepo *repository.AuditRepository,
) *DeviceAPIKeyService {
	return &DeviceAPIKeyService{
		apiKeyRepo: apiKeyRepo,
		roomRepo:   roomRepo,
		auditRepo:  auditRepo,
	}
}

// GenerateAPIKey generates a new API key for a room
// Returns the plain-text key (only shown once) and stores the hashed version
func (s *DeviceAPIKeyService) GenerateAPIKey(roomID uint, description string, userID *uint) (*models.DeviceAPIKeyResponse, error) {
	// Verify room exists
	_, err := s.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return nil, fmt.Errorf("room not found: %w", err)
	}

	// Generate a random 32-byte key
	keyBytes := make([]byte, 32)
	_, err = rand.Read(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Encode to base64 for easier transmission
	plainKey := base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	// Create the API key record
	apiKey := &models.DeviceAPIKey{
		RoomID:      roomID,
		APIKeyHash:  string(hashedKey),
		IsActive:    true,
		Description: description,
	}

	if err := s.apiKeyRepo.CreateAPIKey(apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Audit log
	if userID != nil {
		details := fmt.Sprintf("Generated API key for room_id: %d, description: %s", roomID, description)
		_ = s.auditRepo.CreateAuditLog(userID, "api_key_generate", details)
	}

	// Return the plain key (only time it will be shown)
	return &models.DeviceAPIKeyResponse{
		ID:          apiKey.ID,
		RoomID:      apiKey.RoomID,
		APIKey:      plainKey, // Plain text key
		CreatedAt:   apiKey.CreatedAt,
		ExpiresAt:   apiKey.ExpiresAt,
		IsActive:    apiKey.IsActive,
		Description: apiKey.Description,
	}, nil
}

// ValidateAPIKey validates a plain-text API key for a specific room
func (s *DeviceAPIKeyService) ValidateAPIKey(plainKey string, roomID uint) (bool, error) {
	if plainKey == "" {
		return false, errors.New("API key is required")
	}

	// Get all active API keys for the room
	keys, err := s.apiKeyRepo.GetAPIKeysByRoomID(roomID)
	if err != nil {
		return false, err
	}

	// Try to match the plain key against stored hashes
	for _, key := range keys {
		if !key.IsActive {
			continue
		}

		// Check expiration
		if key.ExpiresAt != nil && key.ExpiresAt.Before(key.CreatedAt) {
			continue
		}

		// Compare the plain key with the hashed key
		err := bcrypt.CompareHashAndPassword([]byte(key.APIKeyHash), []byte(plainKey))
		if err == nil {
			// Key matches!
			return true, nil
		}
	}

	return false, errors.New("invalid API key")
}

// GetAPIKeysByRoomID retrieves all API keys for a room (admin only)
func (s *DeviceAPIKeyService) GetAPIKeysByRoomID(roomID uint, userID uint) ([]models.DeviceAPIKeyResponse, error) {
	// Verify room exists
	_, err := s.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return nil, fmt.Errorf("room not found: %w", err)
	}

	keys, err := s.apiKeyRepo.GetAPIKeysByRoomID(roomID)
	if err != nil {
		return nil, err
	}

	// Convert to response format (without plain keys)
	responses := make([]models.DeviceAPIKeyResponse, len(keys))
	for i, key := range keys {
		responses[i] = models.DeviceAPIKeyResponse{
			ID:          key.ID,
			RoomID:      key.RoomID,
			CreatedAt:   key.CreatedAt,
			ExpiresAt:   key.ExpiresAt,
			IsActive:    key.IsActive,
			Description: key.Description,
			// APIKey is intentionally omitted (never shown after creation)
		}
	}

	return responses, nil
}

// RevokeAPIKey revokes (deactivates) an API key (admin only)
func (s *DeviceAPIKeyService) RevokeAPIKey(keyID uint, userID uint) error {
	// Get the key to verify it exists and for audit logging
	key, err := s.apiKeyRepo.GetAPIKeyByID(keyID)
	if err != nil {
		return err
	}

	// Revoke the key
	if err := s.apiKeyRepo.RevokeAPIKey(keyID); err != nil {
		return err
	}

	// Audit log
	userIDPtr := &userID
	details := fmt.Sprintf("Revoked API key ID: %d for room_id: %d", keyID, key.RoomID)
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "api_key_revoke", details)

	return nil
}

// DeleteAPIKey permanently deletes an API key (admin only)
func (s *DeviceAPIKeyService) DeleteAPIKey(keyID uint, userID uint) error {
	// Get the key to verify it exists and for audit logging
	key, err := s.apiKeyRepo.GetAPIKeyByID(keyID)
	if err != nil {
		return err
	}

	// Delete the key
	if err := s.apiKeyRepo.DeleteAPIKey(keyID); err != nil {
		return err
	}

	// Audit log
	userIDPtr := &userID
	details := fmt.Sprintf("Deleted API key ID: %d for room_id: %d", keyID, key.RoomID)
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "api_key_delete", details)

	return nil
}
