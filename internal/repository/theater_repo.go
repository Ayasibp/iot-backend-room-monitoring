package repository

import (
	"errors"

	"iot-backend-room-monitoring/internal/models"

	"gorm.io/gorm"
)

type TheaterRepository struct {
	db *gorm.DB
}

func NewTheaterRepo(db *gorm.DB) *TheaterRepository {
	return &TheaterRepository{db: db}
}

// GetNewRawLogs fetches raw telemetry logs with ID greater than lastID
// Used by the background worker to poll for new data
// DEPRECATED: Use GetAllRawTelemetry and check updated_at instead
func (r *TheaterRepository) GetNewRawLogs(lastID int) ([]models.TheaterRawTelemetry, error) {
	var logs []models.TheaterRawTelemetry
	err := r.db.Where("id > ?", lastID).
		Order("id ASC").
		Find(&logs).Error
	return logs, err
}

// GetAllRawTelemetry fetches all raw telemetry data for all rooms
func (r *TheaterRepository) GetAllRawTelemetry() ([]models.TheaterRawTelemetry, error) {
	var telemetry []models.TheaterRawTelemetry
	err := r.db.Order("room_name ASC").Find(&telemetry).Error
	return telemetry, err
}

// GetAllLiveStates retrieves live states for all rooms
func (r *TheaterRepository) GetAllLiveStates() ([]models.TheaterLiveState, error) {
	var states []models.TheaterLiveState
	err := r.db.Order("room_name ASC").Find(&states).Error
	return states, err
}

// GetLiveState retrieves the live state for a specific room (legacy method using room_name)
func (r *TheaterRepository) GetLiveState(roomName string) (*models.TheaterLiveState, error) {
	var state models.TheaterLiveState
	err := r.db.Where("room_name = ?", roomName).First(&state).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("live state not found for room: " + roomName)
		}
		return nil, err
	}
	return &state, nil
}

// GetLiveStateByRoomID retrieves the live state for a specific room by room_id
func (r *TheaterRepository) GetLiveStateByRoomID(roomID uint) (*models.TheaterLiveState, error) {
	var state models.TheaterLiveState
	err := r.db.Where("room_id = ?", roomID).Preload("Room.Hospital").First(&state).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("live state not found for room")
		}
		return nil, err
	}
	return &state, nil
}

// UpdateLiveState updates the theater live state
func (r *TheaterRepository) UpdateLiveState(state *models.TheaterLiveState) error {
	return r.db.Save(state).Error
}

// CreateLiveStateIfNotExists creates a live state entry if it doesn't exist
func (r *TheaterRepository) CreateLiveStateIfNotExists(roomName string) error {
	var count int64
	r.db.Model(&models.TheaterLiveState{}).Where("room_name = ?", roomName).Count(&count)

	if count == 0 {
		state := &models.TheaterLiveState{
			RoomName: roomName,
		}
		return r.db.Create(state).Error
	}
	return nil
}

// UpdateOperationTimer updates specific operation timer fields (legacy method using room_name)
func (r *TheaterRepository) UpdateOperationTimer(roomName string, updates map[string]interface{}) error {
	return r.db.Model(&models.TheaterLiveState{}).
		Where("room_name = ?", roomName).
		Updates(updates).Error
}

// UpdateOperationTimerByRoomID updates specific operation timer fields by room_id
func (r *TheaterRepository) UpdateOperationTimerByRoomID(roomID uint, updates map[string]interface{}) error {
	return r.db.Model(&models.TheaterLiveState{}).
		Where("room_id = ?", roomID).
		Updates(updates).Error
}

// UpdateCountdownTimer updates specific countdown timer fields (legacy method using room_name)
func (r *TheaterRepository) UpdateCountdownTimer(roomName string, updates map[string]interface{}) error {
	return r.db.Model(&models.TheaterLiveState{}).
		Where("room_name = ?", roomName).
		Updates(updates).Error
}

// UpdateCountdownTimerByRoomID updates specific countdown timer fields by room_id
func (r *TheaterRepository) UpdateCountdownTimerByRoomID(roomID uint, updates map[string]interface{}) error {
	return r.db.Model(&models.TheaterLiveState{}).
		Where("room_id = ?", roomID).
		Updates(updates).Error
}

// GetRawTelemetryByRoomID retrieves raw telemetry for a specific room
func (r *TheaterRepository) GetRawTelemetryByRoomID(roomID uint) (*models.TheaterRawTelemetry, error) {
	var telemetry models.TheaterRawTelemetry
	err := r.db.Where("room_id = ?", roomID).First(&telemetry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("raw telemetry not found for room")
		}
		return nil, err
	}
	return &telemetry, nil
}
