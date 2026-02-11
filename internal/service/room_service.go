package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type RoomService struct {
	roomRepo         *repository.RoomRepository
	hospitalRepo     *repository.HospitalRepository
	userHospitalRepo *repository.UserHospitalRepository
	auditRepo        *repository.AuditRepository
	theaterRepo      *repository.TheaterRepository
	apiKeyRepo       *repository.DeviceAPIKeyRepository
}

func NewRoomService(
	roomRepo *repository.RoomRepository,
	hospitalRepo *repository.HospitalRepository,
	userHospitalRepo *repository.UserHospitalRepository,
	auditRepo *repository.AuditRepository,
	theaterRepo *repository.TheaterRepository,
	apiKeyRepo *repository.DeviceAPIKeyRepository,
) *RoomService {
	return &RoomService{
		roomRepo:         roomRepo,
		hospitalRepo:     hospitalRepo,
		userHospitalRepo: userHospitalRepo,
		auditRepo:        auditRepo,
		theaterRepo:      theaterRepo,
		apiKeyRepo:       apiKeyRepo,
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

// CreateRoomResponse contains the room and generated API key
type CreateRoomResponse struct {
	Room         *models.Room                 `json:"room"`
	APIKey       *models.DeviceAPIKeyResponse `json:"api_key,omitempty"`
	APIKeyError  string                       `json:"api_key_error,omitempty"`
	TelemetryWarnings []string                `json:"telemetry_warnings,omitempty"`
}

// CreateRoom creates a new room (admin only)
// Automatically initializes telemetry tables and generates an API key
func (s *RoomService) CreateRoom(room *models.Room, userID uint) (*CreateRoomResponse, error) {
	// Verify hospital exists
	_, err := s.hospitalRepo.GetHospitalByID(room.HospitalID)
	if err != nil {
		return nil, fmt.Errorf("hospital not found: %w", err)
	}

	// Create the room
	if err := s.roomRepo.CreateRoom(room); err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	response := &CreateRoomResponse{
		Room:              room,
		TelemetryWarnings: []string{},
	}

	// Initialize theater_raw_telemetry for the new room
	if err := s.theaterRepo.CreateRawTelemetryForRoom(room.ID, room.RoomCode, room.VolumeRuangan); err != nil {
		// Log error but don't fail room creation
		warning := fmt.Sprintf("Failed to create raw telemetry: %v", err)
		fmt.Printf("Warning: %s\n", warning)
		response.TelemetryWarnings = append(response.TelemetryWarnings, warning)
	}

	// Initialize theater_live_state for the new room
	if err := s.theaterRepo.CreateLiveStateForRoom(room.ID, room.RoomCode); err != nil {
		// Log error but don't fail room creation
		warning := fmt.Sprintf("Failed to create live state: %v", err)
		fmt.Printf("Warning: %s\n", warning)
		response.TelemetryWarnings = append(response.TelemetryWarnings, warning)
	}

	// Generate initial API key for ESP32 device
	apiKey, err := s.generateAPIKeyForRoom(room.ID, "Default ESP32 Device", &userID)
	if err != nil {
		// Log error and include in response
		errMsg := fmt.Sprintf("Failed to generate API key: %v. Please run migration: migrations/add_device_api_keys.sql", err)
		fmt.Printf("Warning: %s\n", errMsg)
		response.APIKeyError = errMsg
	} else {
		response.APIKey = apiKey
	}

	// Audit log
	userIDPtr := &userID
	details := fmt.Sprintf("Created room: %s (code: %s, hospital_id: %d)", room.RoomName, room.RoomCode, room.HospitalID)
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "room_create", details)

	return response, nil
}

// generateAPIKeyForRoom is a helper method to generate an API key for a room
func (s *RoomService) generateAPIKeyForRoom(roomID uint, description string, userID *uint) (*models.DeviceAPIKeyResponse, error) {
	// Generate a random 32-byte key
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Encode to base64 for easier transmission
	plainKey := base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	// Create the API key record
	apiKey := &models.DeviceAPIKey{
		RoomID:      roomID,
		APIKeyHash:  string(hashedKey),
		IsActive:    true,
		Description: description,
	}

	if err := s.apiKeyRepo.CreateAPIKey(apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Return the plain key (only time it will be shown)
	return &models.DeviceAPIKeyResponse{
		ID:          apiKey.ID,
		RoomID:      apiKey.RoomID,
		APIKey:      plainKey, // Plain text key
		CreatedAt:   apiKey.CreatedAt,
		ExpiresAt:   apiKey.ExpiresAt,
		IsActive:    apiKey.IsActive,
		Description: apiKey.Description,
	}, nil
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
