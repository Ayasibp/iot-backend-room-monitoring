package service

import (
	"errors"
	"fmt"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/repository"
)

type RoomService struct {
	roomRepo         *repository.RoomRepository
	hospitalRepo     *repository.HospitalRepository
	userHospitalRepo *repository.UserHospitalRepository
	auditRepo        *repository.AuditRepository
}

func NewRoomService(
	roomRepo *repository.RoomRepository,
	hospitalRepo *repository.HospitalRepository,
	userHospitalRepo *repository.UserHospitalRepository,
	auditRepo *repository.AuditRepository,
) *RoomService {
	return &RoomService{
		roomRepo:         roomRepo,
		hospitalRepo:     hospitalRepo,
		userHospitalRepo: userHospitalRepo,
		auditRepo:        auditRepo,
	}
}

// GetRoomsByHospitalID retrieves all rooms for a hospital with access control
func (s *RoomService) GetRoomsByHospitalID(hospitalID uint, userID uint, role string) ([]models.Room, error) {
	// Check hospital access
	if err := s.checkHospitalAccess(hospitalID, userID, role); err != nil {
		return nil, err
	}

	return s.roomRepo.GetRoomsByHospitalID(hospitalID)
}

// GetRoomByID retrieves a room by ID with access control
func (s *RoomService) GetRoomByID(roomID uint, userID uint, role string) (*models.Room, error) {
	// Get room with hospital information
	room, err := s.roomRepo.GetRoomWithHospital(roomID)
	if err != nil {
		return nil, err
	}

	// Check hospital access
	if err := s.checkHospitalAccess(room.HospitalID, userID, role); err != nil {
		return nil, err
	}

	return room, nil
}

// CreateRoom creates a new room (admin only)
func (s *RoomService) CreateRoom(room *models.Room, userID uint) error {
	// Verify hospital exists
	_, err := s.hospitalRepo.GetHospitalByID(room.HospitalID)
	if err != nil {
		return fmt.Errorf("hospital not found: %w", err)
	}

	// Create the room
	if err := s.roomRepo.CreateRoom(room); err != nil {
		return fmt.Errorf("failed to create room: %w", err)
	}

	// Audit log
	userIDPtr := &userID
	details := fmt.Sprintf("Created room: %s (code: %s, hospital_id: %d)", room.RoomName, room.RoomCode, room.HospitalID)
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "room_create", details)

	return nil
}

// UpdateRoom updates an existing room (admin only)
func (s *RoomService) UpdateRoom(room *models.Room, userID uint) error {
	// Verify room exists
	existing, err := s.roomRepo.GetRoomByID(room.ID)
	if err != nil {
		return err
	}

	// Verify hospital exists if hospital_id is being changed
	if room.HospitalID != existing.HospitalID {
		_, err := s.hospitalRepo.GetHospitalByID(room.HospitalID)
		if err != nil {
			return fmt.Errorf("hospital not found: %w", err)
		}
	}

	// Update the room
	if err := s.roomRepo.UpdateRoom(room); err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}

	// Audit log
	userIDPtr := &userID
	details := fmt.Sprintf("Updated room: %s (ID: %d, code: %s)", room.RoomName, room.ID, room.RoomCode)
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "room_update", details)

	return nil
}

// DeleteRoom soft deletes a room (admin only)
func (s *RoomService) DeleteRoom(roomID uint, userID uint) error {
	// Verify room exists
	room, err := s.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return err
	}

	// Soft delete
	if err := s.roomRepo.SoftDeleteRoom(roomID); err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}

	// Audit log
	userIDPtr := &userID
	details := fmt.Sprintf("Deleted room: %s (code: %s, ID: %d)", room.RoomName, room.RoomCode, roomID)
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "room_delete", details)

	return nil
}

// GetAllRoomsByUser retrieves all rooms accessible by a user
func (s *RoomService) GetAllRoomsByUser(userID uint, role string) ([]models.Room, error) {
	if role == "admin" {
		return s.roomRepo.GetAllRooms()
	}
	return s.roomRepo.GetRoomsByUserID(userID)
}

// CheckUserRoomAccess checks if a user has access to a specific room
func (s *RoomService) CheckUserRoomAccess(roomID uint, userID uint, role string) error {
	// Admin users have access to all rooms
	if role == "admin" {
		return nil
	}

	// Get room to find its hospital
	room, err := s.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return err
	}

	// Check hospital access
	return s.checkHospitalAccess(room.HospitalID, userID, role)
}

// checkHospitalAccess is a helper method to verify hospital access
func (s *RoomService) checkHospitalAccess(hospitalID uint, userID uint, role string) error {
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
		return errors.New("access denied: you don't have permission to access this hospital's rooms")
	}

	return nil
}
