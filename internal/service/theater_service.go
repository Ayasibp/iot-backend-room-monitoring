package service

import (
	"errors"
	"fmt"
	"time"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/repository"
)

type TheaterService struct {
	theaterRepo      *repository.TheaterRepository
	auditRepo        *repository.AuditRepository
	roomRepo         *repository.RoomRepository
	userHospitalRepo *repository.UserHospitalRepository
}

func NewTheaterService(
	theaterRepo *repository.TheaterRepository,
	auditRepo *repository.AuditRepository,
) *TheaterService {
	return &TheaterService{
		theaterRepo: theaterRepo,
		auditRepo:   auditRepo,
	}
}

// NewTheaterServiceWithAccessControl creates a theater service with access control support
func NewTheaterServiceWithAccessControl(
	theaterRepo *repository.TheaterRepository,
	auditRepo *repository.AuditRepository,
	roomRepo *repository.RoomRepository,
	userHospitalRepo *repository.UserHospitalRepository,
) *TheaterService {
	return &TheaterService{
		theaterRepo:      theaterRepo,
		auditRepo:        auditRepo,
		roomRepo:         roomRepo,
		userHospitalRepo: userHospitalRepo,
	}
}

// GetLiveState retrieves the live state for a room
func (s *TheaterService) GetLiveState(roomName string) (*models.TheaterLiveState, error) {
	return s.theaterRepo.GetLiveState(roomName)
}

// GetAllLiveStates retrieves live states for all rooms
func (s *TheaterService) GetAllLiveStates() ([]models.TheaterLiveState, error) {
	return s.theaterRepo.GetAllLiveStates()
}

// GetAllRooms retrieves a list of all room names
func (s *TheaterService) GetAllRooms() ([]string, error) {
	states, err := s.theaterRepo.GetAllLiveStates()
	if err != nil {
		return nil, err
	}

	rooms := make([]string, len(states))
	for i, state := range states {
		rooms[i] = state.RoomName
	}
	return rooms, nil
}

// UpdateOperationTimer handles start/stop/reset actions for operation timer
func (s *TheaterService) UpdateOperationTimer(roomName, action string, userID uint) error {
	// Get current state
	state, err := s.theaterRepo.GetLiveState(roomName)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})
	var auditDetails string

	switch action {
	case "start":
		if state.OpIsRunning {
			return errors.New("timer is already running")
		}
		now := time.Now()
		updates["op_start_time"] = now
		updates["op_is_running"] = true
		auditDetails = fmt.Sprintf("Started operation timer for room %s", roomName)

	case "stop":
		if !state.OpIsRunning {
			return errors.New("timer is not running")
		}
		if state.OpStartTime != nil {
			elapsed := int(time.Since(*state.OpStartTime).Seconds())
			updates["op_accumulated_seconds"] = state.OpAccumulatedSeconds + elapsed
		}
		updates["op_is_running"] = false
		auditDetails = fmt.Sprintf("Stopped operation timer for room %s", roomName)

	case "reset":
		updates["op_start_time"] = nil
		updates["op_accumulated_seconds"] = 0
		updates["op_is_running"] = false
		auditDetails = fmt.Sprintf("Reset operation timer for room %s", roomName)

	default:
		return errors.New("invalid action: must be 'start', 'stop', or 'reset'")
	}

	// Update the state
	if err := s.theaterRepo.UpdateOperationTimer(roomName, updates); err != nil {
		return fmt.Errorf("failed to update operation timer: %w", err)
	}

	// Log the action
	userIDPtr := &userID
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "timer_operation", auditDetails)

	return nil
}

// UpdateCountdownTimer handles start/stop/reset actions for countdown timer
func (s *TheaterService) UpdateCountdownTimer(roomName, action string, durationMinutes *int, userID uint) error {
	// Get current state
	state, err := s.theaterRepo.GetLiveState(roomName)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})
	var auditDetails string

	switch action {
	case "start":
		if state.CdIsRunning {
			return errors.New("countdown timer is already running")
		}
		
		// Default duration is 60 minutes if not specified
		duration := 60
		if durationMinutes != nil && *durationMinutes > 0 {
			duration = *durationMinutes
		}
		
		targetTime := time.Now().Add(time.Duration(duration) * time.Minute)
		updates["cd_target_time"] = targetTime
		updates["cd_duration_seconds"] = duration * 60
		updates["cd_is_running"] = true
		auditDetails = fmt.Sprintf("Started countdown timer for room %s with duration %d minutes", roomName, duration)

	case "stop":
		if !state.CdIsRunning {
			return errors.New("countdown timer is not running")
		}
		updates["cd_is_running"] = false
		auditDetails = fmt.Sprintf("Stopped countdown timer for room %s", roomName)

	case "reset":
		updates["cd_target_time"] = nil
		updates["cd_duration_seconds"] = 3600
		updates["cd_is_running"] = false
		auditDetails = fmt.Sprintf("Reset countdown timer for room %s", roomName)

	default:
		return errors.New("invalid action: must be 'start', 'stop', or 'reset'")
	}

	// Update the state
	if err := s.theaterRepo.UpdateCountdownTimer(roomName, updates); err != nil {
		return fmt.Errorf("failed to update countdown timer: %w", err)
	}

	// Log the action
	userIDPtr := &userID
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "countdown_timer_operation", auditDetails)

	return nil
}

// AdjustCountdownTimer adjusts the countdown timer by adding or subtracting minutes
func (s *TheaterService) AdjustCountdownTimer(roomName string, minutes int, userID uint) error {
	// Get current state
	state, err := s.theaterRepo.GetLiveState(roomName)
	if err != nil {
		return err
	}

	// Validate timer is running
	if !state.CdIsRunning {
		return errors.New("countdown timer is not running")
	}

	// Validate target time exists
	if state.CdTargetTime == nil {
		return errors.New("countdown timer has no target time set")
	}

	// Calculate new target time
	newTargetTime := state.CdTargetTime.Add(time.Duration(minutes) * time.Minute)
	
	// Validate new target time is not in the past
	now := time.Now()
	if newTargetTime.Before(now) {
		// If adjustment would make it negative, set to current time + 1 second
		newTargetTime = now.Add(1 * time.Second)
	}

	// Update the target time
	updates := map[string]interface{}{
		"cd_target_time": newTargetTime,
	}

	if err := s.theaterRepo.UpdateCountdownTimer(roomName, updates); err != nil {
		return fmt.Errorf("failed to adjust countdown timer: %w", err)
	}

	// Log the action
	userIDPtr := &userID
	action := "increased"
	if minutes < 0 {
		action = "decreased"
	}
	auditDetails := fmt.Sprintf("Adjusted countdown timer for room %s: %s by %d minute(s)", roomName, action, abs(minutes))
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "countdown_timer_adjustment", auditDetails)

	return nil
}

// abs returns the absolute value of an integer
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// GetLiveStateByRoomID retrieves the live state for a room by room_id
func (s *TheaterService) GetLiveStateByRoomID(roomID uint, userID uint, role string) (*models.TheaterLiveState, error) {
	// Check access control if repositories are configured
	if err := s.checkUserRoomAccess(roomID, userID, role); err != nil {
		return nil, err
	}

	return s.theaterRepo.GetLiveStateByRoomID(roomID)
}

// UpdateOperationTimerByRoomID handles start/stop/reset actions for operation timer by room_id
func (s *TheaterService) UpdateOperationTimerByRoomID(roomID uint, action string, userID uint, role string) error {
	// Check access control
	if err := s.checkUserRoomAccess(roomID, userID, role); err != nil {
		return err
	}

	// Get current state
	state, err := s.theaterRepo.GetLiveStateByRoomID(roomID)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})
	var auditDetails string

	switch action {
	case "start":
		if state.OpIsRunning {
			return errors.New("timer is already running")
		}
		now := time.Now()
		updates["op_start_time"] = now
		updates["op_is_running"] = true
		auditDetails = fmt.Sprintf("Started operation timer for room_id %d", roomID)

	case "stop":
		if !state.OpIsRunning {
			return errors.New("timer is not running")
		}
		if state.OpStartTime != nil {
			elapsed := int(time.Since(*state.OpStartTime).Seconds())
			updates["op_accumulated_seconds"] = state.OpAccumulatedSeconds + elapsed
		}
		updates["op_is_running"] = false
		auditDetails = fmt.Sprintf("Stopped operation timer for room_id %d", roomID)

	case "reset":
		updates["op_start_time"] = nil
		updates["op_accumulated_seconds"] = 0
		updates["op_is_running"] = false
		auditDetails = fmt.Sprintf("Reset operation timer for room_id %d", roomID)

	default:
		return errors.New("invalid action: must be 'start', 'stop', or 'reset'")
	}

	// Update the state
	if err := s.theaterRepo.UpdateOperationTimerByRoomID(roomID, updates); err != nil {
		return fmt.Errorf("failed to update operation timer: %w", err)
	}

	// Log the action
	userIDPtr := &userID
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "timer_operation", auditDetails)

	return nil
}

// UpdateCountdownTimerByRoomID handles start/stop/reset actions for countdown timer by room_id
func (s *TheaterService) UpdateCountdownTimerByRoomID(roomID uint, action string, durationMinutes *int, userID uint, role string) error {
	// Check access control
	if err := s.checkUserRoomAccess(roomID, userID, role); err != nil {
		return err
	}

	// Get current state
	state, err := s.theaterRepo.GetLiveStateByRoomID(roomID)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})
	var auditDetails string

	switch action {
	case "start":
		if state.CdIsRunning {
			return errors.New("countdown timer is already running")
		}

		// Default duration is 60 minutes if not specified
		duration := 60
		if durationMinutes != nil && *durationMinutes > 0 {
			duration = *durationMinutes
		}

		targetTime := time.Now().Add(time.Duration(duration) * time.Minute)
		updates["cd_target_time"] = targetTime
		updates["cd_duration_seconds"] = duration * 60
		updates["cd_is_running"] = true
		auditDetails = fmt.Sprintf("Started countdown timer for room_id %d with duration %d minutes", roomID, duration)

	case "stop":
		if !state.CdIsRunning {
			return errors.New("countdown timer is not running")
		}
		updates["cd_is_running"] = false
		auditDetails = fmt.Sprintf("Stopped countdown timer for room_id %d", roomID)

	case "reset":
		updates["cd_target_time"] = nil
		updates["cd_duration_seconds"] = 3600
		updates["cd_is_running"] = false
		auditDetails = fmt.Sprintf("Reset countdown timer for room_id %d", roomID)

	default:
		return errors.New("invalid action: must be 'start', 'stop', or 'reset'")
	}

	// Update the state
	if err := s.theaterRepo.UpdateCountdownTimerByRoomID(roomID, updates); err != nil {
		return fmt.Errorf("failed to update countdown timer: %w", err)
	}

	// Log the action
	userIDPtr := &userID
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "countdown_timer_operation", auditDetails)

	return nil
}

// AdjustCountdownTimerByRoomID adjusts the countdown timer by adding or subtracting minutes by room_id
func (s *TheaterService) AdjustCountdownTimerByRoomID(roomID uint, minutes int, userID uint, role string) error {
	// Check access control
	if err := s.checkUserRoomAccess(roomID, userID, role); err != nil {
		return err
	}

	// Get current state
	state, err := s.theaterRepo.GetLiveStateByRoomID(roomID)
	if err != nil {
		return err
	}

	// Validate timer is running
	if !state.CdIsRunning {
		return errors.New("countdown timer is not running")
	}

	// Validate target time exists
	if state.CdTargetTime == nil {
		return errors.New("countdown timer has no target time set")
	}

	// Calculate new target time
	newTargetTime := state.CdTargetTime.Add(time.Duration(minutes) * time.Minute)

	// Validate new target time is not in the past
	now := time.Now()
	if newTargetTime.Before(now) {
		// If adjustment would make it negative, set to current time + 1 second
		newTargetTime = now.Add(1 * time.Second)
	}

	// Update the target time
	updates := map[string]interface{}{
		"cd_target_time": newTargetTime,
	}

	if err := s.theaterRepo.UpdateCountdownTimerByRoomID(roomID, updates); err != nil {
		return fmt.Errorf("failed to adjust countdown timer: %w", err)
	}

	// Log the action
	userIDPtr := &userID
	action := "increased"
	if minutes < 0 {
		action = "decreased"
	}
	auditDetails := fmt.Sprintf("Adjusted countdown timer for room_id %d: %s by %d minute(s)", roomID, action, abs(minutes))
	_ = s.auditRepo.CreateAuditLog(userIDPtr, "countdown_timer_adjustment", auditDetails)

	return nil
}

// checkUserRoomAccess checks if a user has access to a specific room
func (s *TheaterService) checkUserRoomAccess(roomID uint, userID uint, role string) error {
	// If access control dependencies are not configured, skip the check
	if s.roomRepo == nil || s.userHospitalRepo == nil {
		return nil
	}

	// Admin users have access to all rooms
	if role == "admin" {
		return nil
	}

	// Get room to find its hospital
	room, err := s.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return err
	}

	// Check if user has access to the room's hospital
	hasAccess, err := s.userHospitalRepo.UserHasAccessToHospital(userID, room.HospitalID)
	if err != nil {
		return err
	}
	if !hasAccess {
		return errors.New("access denied: you don't have permission to access this room")
	}

	return nil
}
