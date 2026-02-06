package models

import "time"

// User represents the users table
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null;size:50" json:"username"`
	PasswordHash string    `gorm:"not null;size:255" json:"-"`
	Role         string    `gorm:"type:enum('admin','user');default:'user'" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}

// RefreshToken represents the refresh_tokens table
type RefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	TokenHash string    `gorm:"not null;size:255;index" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `gorm:"default:false" json:"revoked"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName specifies the table name for RefreshToken model
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
