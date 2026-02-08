package service

import (
	"errors"
	"fmt"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/repository"
)

type HospitalService struct {
	hospitalRepo     *repository.HospitalRepository
	userHospitalRepo *repository.UserHospitalRepository
	auditRepo        *repository.AuditRepository
}

func NewHospitalService(
	hospitalRepo *repository.HospitalRepository,
	userHospitalRepo *repository.UserHospitalRepository,
	auditRepo *repository.AuditRepository,
) *HospitalService {
	return &HospitalService{
		hospitalRepo:     hospitalRepo,
		userHospitalRepo: userHospitalRepo,
		auditRepo:        auditRepo,
	}
}

// GetAllHospitals retrieves hospitals based on user role
// Admin users see all hospitals, regular users see only assigned hospitals
func (s *HospitalService) GetAllHospitals(userID uint, role string) ([]models.Hospital, error) {
	if role == "admin" {
		return s.hospitalRepo.GetAllHospitals()
	}
	return s.hospitalRepo.GetHospitalsByUserID(userID)
}

// GetHospitalByID retrieves a hospital by ID with access control
func (s *HospitalService) GetHospitalByID(id uint, userID uint, role string) (*models.Hospital, error) {
	// Admin users can access any hospital
	if role == "admin" {
		return s.hospitalRepo.GetHospitalByID(id)
	}

	// Regular users must have explicit access
	hasAccess, err := s.userHospitalRepo.UserHasAccessToHospital(userID, id)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, errors.New("access denied: you don't have permission to view this hospital")
	}

	return s.hospitalRepo.GetHospitalByID(id)
}

// CreateHospital creates a new hospital (admin only)
func (s *HospitalService) CreateHospital(hospital *models.Hospital, userID uint) error {
	// Create the hospital
	if err := s.hospitalRepo.CreateHospital(hospital); err != nil {
		return fmt.Errorf("failed to create hospital: %w", err)
	}

	// Automatically assign all admin users to the new hospital
	// This is handled separately via AssignAdminToHospital

	// Audit log
	userIDPtr := &userID
	details := fmt.Sprintf("Created hospital: %s (code: %s)", hospital.Name, hospital.Code)
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "hospital_create", details)

	return nil
}

// UpdateHospital updates an existing hospital (admin only)
func (s *HospitalService) UpdateHospital(hospital *models.Hospital, userID uint) error {
	// Verify hospital exists
	existing, err := s.hospitalRepo.GetHospitalByID(hospital.ID)
	if err != nil {
		return err
	}

	// Update the hospital
	if err := s.hospitalRepo.UpdateHospital(hospital); err != nil {
		return fmt.Errorf("failed to update hospital: %w", err)
	}

	// Audit log
	userIDPtr := &userID
	details := fmt.Sprintf("Updated hospital: %s (ID: %d, old code: %s)", hospital.Name, hospital.ID, existing.Code)
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "hospital_update", details)

	return nil
}

// DeleteHospital soft deletes a hospital (admin only)
func (s *HospitalService) DeleteHospital(id uint, userID uint) error {
	// Verify hospital exists
	hospital, err := s.hospitalRepo.GetHospitalByID(id)
	if err != nil {
		return err
	}

	// Soft delete
	if err := s.hospitalRepo.SoftDeleteHospital(id); err != nil {
		return fmt.Errorf("failed to delete hospital: %w", err)
	}

	// Audit log
	userIDPtr := &userID
	details := fmt.Sprintf("Deleted hospital: %s (code: %s, ID: %d)", hospital.Name, hospital.Code, id)
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "hospital_delete", details)

	return nil
}

// AssignUserToHospital assigns a user to a hospital (admin only)
func (s *HospitalService) AssignUserToHospital(userID uint, hospitalID uint, adminUserID uint) error {
	// Verify hospital exists
	_, err := s.hospitalRepo.GetHospitalByID(hospitalID)
	if err != nil {
		return err
	}

	// Assign user to hospital
	if err := s.userHospitalRepo.AssignUserToHospital(userID, hospitalID); err != nil {
		return fmt.Errorf("failed to assign user to hospital: %w", err)
	}

	// Audit log
	adminUserIDPtr := &adminUserID
	details := fmt.Sprintf("Assigned user ID %d to hospital ID %d", userID, hospitalID)
	_ = s.auditRepo.CreateAuditLog(adminUserIDPtr, "user_hospital_assign", details)

	return nil
}

// RemoveUserFromHospital removes a user's access to a hospital (admin only)
func (s *HospitalService) RemoveUserFromHospital(userID uint, hospitalID uint, adminUserID uint) error {
	// Remove assignment
	if err := s.userHospitalRepo.RemoveUserFromHospital(userID, hospitalID); err != nil {
		return fmt.Errorf("failed to remove user from hospital: %w", err)
	}

	// Audit log
	adminUserIDPtr := &adminUserID
	details := fmt.Sprintf("Removed user ID %d from hospital ID %d", userID, hospitalID)
	_ = s.auditRepo.CreateAuditLog(adminUserIDPtr, "user_hospital_remove", details)

	return nil
}

// CheckUserHospitalAccess checks if a user has access to a hospital
func (s *HospitalService) CheckUserHospitalAccess(userID uint, hospitalID uint, role string) error {
	// Admin users have access to all hospitals
	if role == "admin" {
		return nil
	}

	// Regular users must have explicit access
	hasAccess, err := s.userHospitalRepo.UserHasAccessToHospital(userID, hospitalID)
	if err != nil {
		return err
	}
	if !hasAccess {
		return errors.New("access denied: you don't have permission to access this hospital")
	}

	return nil
}
