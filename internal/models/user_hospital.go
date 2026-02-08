package models

import "time"

// UserHospital represents the many-to-many relationship between users and hospitals
// This table controls which hospitals a user has access to
type UserHospital struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	HospitalID uint      `gorm:"not null;index" json:"hospital_id"`
	CreatedAt  time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`

	// Relationships
	User     User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Hospital Hospital `gorm:"foreignKey:HospitalID" json:"hospital,omitempty"`
}

// TableName specifies the table name for UserHospital model
func (UserHospital) TableName() string {
	return "user_hospitals"
}
