package service

import (
	"errors"
	"fmt"
	"time"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/repository"
)

type TheaterService struct {
	theaterRepo *repository.TheaterRepository
	auditRepo   *repository.AuditRepository
}

func NewTheaterService(theaterRepo *repository.TheaterRepository, auditRepo *repository.AuditRepository) *TheaterService {
	return &TheaterService{
		theaterRepo: theaterRepo,
		auditRepo:   auditRepo,
	}
}

// GetLiveState retrieves the live state for a room
func (s *TheaterService) GetLiveState(roomName string) (*models.TheaterLiveState, error) {
	return s.theaterRepo.GetLiveState(roomName)
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
