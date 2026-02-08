package repository

import (
	"iot-backend-room-monitoring/internal/models"

	"gorm.io/gorm"
)

type UserHospitalRepository struct {
	db *gorm.DB
}

func NewUserHospitalRepo(db *gorm.DB) *UserHospitalRepository {
	return &UserHospitalRepository{db: db}
}

// AssignUserToHospital assigns a user to a hospital
func (r *UserHospitalRepository) AssignUserToHospital(userID, hospitalID uint) error {
	userHospital := &models.UserHospital{
		UserID:     userID,
		HospitalID: hospitalID,
	}
	// Use FirstOrCreate to avoid duplicate entries
	return r.db.Where("user_id = ? AND hospital_id = ?", userID, hospitalID).
		FirstOrCreate(userHospital).Error
}

// RemoveUserFromHospital removes a user's access to a hospital
func (r *UserHospitalRepository) RemoveUserFromHospital(userID, hospitalID uint) error {
	return r.db.Where("user_id = ? AND hospital_id = ?", userID, hospitalID).
		Delete(&models.UserHospital{}).Error
}

// GetUserHospitals retrieves all hospital IDs a user has access to
func (r *UserHospitalRepository) GetUserHospitals(userID uint) ([]uint, error) {
	var hospitalIDs []uint
	err := r.db.Model(&models.UserHospital{}).
		Where("user_id = ?", userID).
		Pluck("hospital_id", &hospitalIDs).Error
	return hospitalIDs, err
}

// GetHospitalUsers retrieves all user IDs that have access to a hospital
func (r *UserHospitalRepository) GetHospitalUsers(hospitalID uint) ([]uint, error) {
	var userIDs []uint
	err := r.db.Model(&models.UserHospital{}).
		Where("hospital_id = ?", hospitalID).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// UserHasAccessToHospital checks if a user has access to a specific hospital
func (r *UserHospitalRepository) UserHasAccessToHospital(userID, hospitalID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.UserHospital{}).
		Where("user_id = ? AND hospital_id = ?", userID, hospitalID).
		Count(&count).Error
	return count > 0, err
}

// AssignUserToAllHospitals assigns a user to all active hospitals (typically for admins)
func (r *UserHospitalRepository) AssignUserToAllHospitals(userID uint) error {
	// Get all active hospital IDs
	var hospitalIDs []uint
	err := r.db.Model(&models.Hospital{}).
		Where("is_active = ?", true).
		Pluck("id", &hospitalIDs).Error
	if err != nil {
		return err
	}

	// Assign user to each hospital
	for _, hospitalID := range hospitalIDs {
		if err := r.AssignUserToHospital(userID, hospitalID); err != nil {
			return err
		}
	}
	return nil
}
