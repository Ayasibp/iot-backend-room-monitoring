package models

import "time"

// Hospital represents a hospital/medical facility in the system
type Hospital struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:50;uniqueIndex" json:"code"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Address   string    `gorm:"type:text" json:"address,omitempty"`
	City      string    `gorm:"size:100" json:"city,omitempty"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
}

// TableName specifies the table name for Hospital model
func (Hospital) TableName() string {
	return "hospitals"
}
