package repository

import (
	"errors"
	"time"

	"iot-backend-room-monitoring/internal/models"

	"gorm.io/gorm"
)

type DeviceAPIKeyRepository struct {
	db *gorm.DB
}

func NewDeviceAPIKeyRepo(db *gorm.DB) *DeviceAPIKeyRepository {
	return &DeviceAPIKeyRepository{db: db}
}

// CreateAPIKey creates a new API key for a room
func (r *DeviceAPIKeyRepository) CreateAPIKey(key *models.DeviceAPIKey) error {
	return r.db.Create(key).Error
}

// GetAPIKeyByHash retrieves an API key by its hash
func (r *DeviceAPIKeyRepository) GetAPIKeyByHash(keyHash string) (*models.DeviceAPIKey, error) {
	var key models.DeviceAPIKey
	err := r.db.Where("api_key_hash = ? AND is_active = ?", keyHash, true).First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API key not found or inactive")
		}
		return nil, err
	}

	// Check if key has expired
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("API key has expired")
	}

	return &key, nil
}

// ValidateAPIKey validates if an API key hash is valid for a specific room
func (r *DeviceAPIKeyRepository) ValidateAPIKey(keyHash string, roomID uint) (bool, error) {
	key, err := r.GetAPIKeyByHash(keyHash)
	if err != nil {
		return false, err
	}

	// Verify the key belongs to the specified room
	if key.RoomID != roomID {
		return false, errors.New("API key does not belong to this room")
	}

	return true, nil
}

// GetAPIKeysByRoomID retrieves all API keys for a specific room
func (r *DeviceAPIKeyRepository) GetAPIKeysByRoomID(roomID uint) ([]models.DeviceAPIKey, error) {
	var keys []models.DeviceAPIKey
	err := r.db.Where("room_id = ?", roomID).
		Order("created_at DESC").
		Find(&keys).Error
	return keys, err
}

// GetAPIKeyByID retrieves an API key by its ID
func (r *DeviceAPIKeyRepository) GetAPIKeyByID(keyID uint) (*models.DeviceAPIKey, error) {
	var key models.DeviceAPIKey
	err := r.db.Where("id = ?", keyID).First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API key not found")
		}
		return nil, err
	}
	return &key, nil
}

// RevokeAPIKey revokes (deactivates) an API key
func (r *DeviceAPIKeyRepository) RevokeAPIKey(keyID uint) error {
	result := r.db.Model(&models.DeviceAPIKey{}).
		Where("id = ?", keyID).
		Update("is_active", false)
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return errors.New("API key not found")
	}
	
	return nil
}

// DeleteAPIKey permanently deletes an API key
func (r *DeviceAPIKeyRepository) DeleteAPIKey(keyID uint) error {
	result := r.db.Delete(&models.DeviceAPIKey{}, keyID)
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return errors.New("API key not found")
	}
	
	return nil
}

// GetActiveAPIKeysCount returns the count of active API keys for a room
func (r *DeviceAPIKeyRepository) GetActiveAPIKeysCount(roomID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.DeviceAPIKey{}).
		Where("room_id = ? AND is_active = ?", roomID, true).
		Count(&count).Error
	return count, err
}
