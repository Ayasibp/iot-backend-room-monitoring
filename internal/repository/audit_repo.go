package repository

import (
	"iot-backend-room-monitoring/internal/models"

	"gorm.io/gorm"
)

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// CreateAuditLog creates a new audit log entry
func (r *AuditRepository) CreateAuditLog(userID *uint, action string, details string) error {
	log := &models.AuditLog{
		UserID:  userID,
		Action:  action,
		Details: details,
	}
	return r.db.Create(log).Error
}
