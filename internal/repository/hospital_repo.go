package repository

import (
	"errors"
	"iot-backend-room-monitoring/internal/models"

	"gorm.io/gorm"
)

type HospitalRepository struct {
	db *gorm.DB
}

func NewHospitalRepo(db *gorm.DB) *HospitalRepository {
	return &HospitalRepository{db: db}
}

// GetAllHospitals retrieves all active hospitals
func (r *HospitalRepository) GetAllHospitals() ([]models.Hospital, error) {
	var hospitals []models.Hospital
	err := r.db.Where("is_active = ?", true).Order("name ASC").Find(&hospitals).Error
	return hospitals, err
}

// GetHospitalByID retrieves a hospital by ID
func (r *HospitalRepository) GetHospitalByID(id uint) (*models.Hospital, error) {
	var hospital models.Hospital
	err := r.db.Where("id = ? AND is_active = ?", id, true).First(&hospital).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("hospital not found")
		}
		return nil, err
	}
	return &hospital, nil
}

// GetHospitalsByUserID retrieves hospitals accessible by a specific user
// Joins with user_hospitals table to filter by user access
func (r *HospitalRepository) GetHospitalsByUserID(userID uint) ([]models.Hospital, error) {
	var hospitals []models.Hospital
	err := r.db.
		Joins("INNER JOIN user_hospitals ON user_hospitals.hospital_id = hospitals.id").
		Where("user_hospitals.user_id = ? AND hospitals.is_active = ?", userID, true).
		Order("hospitals.name ASC").
		Find(&hospitals).Error
	return hospitals, err
}

// CreateHospital creates a new hospital
func (r *HospitalRepository) CreateHospital(hospital *models.Hospital) error {
	return r.db.Create(hospital).Error
}

// UpdateHospital updates an existing hospital
func (r *HospitalRepository) UpdateHospital(hospital *models.Hospital) error {
	return r.db.Save(hospital).Error
}

// SoftDeleteHospital soft deletes a hospital by setting is_active to false
func (r *HospitalRepository) SoftDeleteHospital(id uint) error {
	return r.db.Model(&models.Hospital{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// GetHospitalByCode retrieves a hospital by its unique code
func (r *HospitalRepository) GetHospitalByCode(code string) (*models.Hospital, error) {
	var hospital models.Hospital
	err := r.db.Where("code = ? AND is_active = ?", code, true).First(&hospital).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("hospital not found")
		}
		return nil, err
	}
	return &hospital, nil
}
