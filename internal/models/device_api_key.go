package models

import "time"

// DeviceAPIKey represents the device_api_keys table
// Used for authenticating ESP32 devices that send telemetry data
type DeviceAPIKey struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	RoomID      uint       `gorm:"not null;index" json:"room_id"`
	APIKeyHash  string     `gorm:"size:255;not null;uniqueIndex" json:"-"` // Hidden from JSON for security
	CreatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	Description string     `gorm:"size:255" json:"description,omitempty"`

	// Relationships
	Room Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

// TableName specifies the table name for DeviceAPIKey model
func (DeviceAPIKey) TableName() string {
	return "device_api_keys"
}

// DeviceAPIKeyResponse is used when returning API keys to the client
// Includes the plain-text key (only shown once during generation)
type DeviceAPIKeyResponse struct {
	ID          uint       `json:"id"`
	RoomID      uint       `json:"room_id"`
	APIKey      string     `json:"api_key,omitempty"` // Plain-text key, only populated during generation
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool       `json:"is_active"`
	Description string     `json:"description,omitempty"`
}
