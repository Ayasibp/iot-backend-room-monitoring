package models

import "time"

// Room represents a room (e.g., operating theater, ICU) within a hospital
type Room struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	HospitalID    uint      `gorm:"not null;index" json:"hospital_id"`
	RoomCode      string    `gorm:"size:50;not null" json:"room_code"`
	RoomName      string    `gorm:"size:100;not null" json:"room_name"`
	RoomType      string    `gorm:"type:enum('operating_theater','icu','isolation','general');default:'operating_theater'" json:"room_type"`
	VolumeRuangan int       `gorm:"default:0;comment:Room volume for ACH calculation" json:"volume_ruangan"`
	CreatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`

	// Relationships
	Hospital Hospital `gorm:"foreignKey:HospitalID" json:"hospital,omitempty"`
}

// TableName specifies the table name for Room model
func (Room) TableName() string {
	return "rooms"
}

// RoomWithDetails includes hospital information for dashboard display
type RoomWithDetails struct {
	Room
	HospitalCode string `json:"hospital_code"`
	HospitalName string `json:"hospital_name"`
}
