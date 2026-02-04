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
func (r *TheaterRepository) GetNewRawLogs(lastID int) ([]models.TheaterRawTelemetry, error) {
	var logs []models.TheaterRawTelemetry
	err := r.db.Where("id > ?", lastID).
		Order("id ASC").
		Find(&logs).Error
	return logs, err
}

// GetLiveState retrieves the live state for a specific room
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

// UpdateOperationTimer updates specific operation timer fields
func (r *TheaterRepository) UpdateOperationTimer(roomName string, updates map[string]interface{}) error {
	return r.db.Model(&models.TheaterLiveState{}).
		Where("room_name = ?", roomName).
		Updates(updates).Error
}
