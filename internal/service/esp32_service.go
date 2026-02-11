package service

import (
	"errors"
	"fmt"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/repository"
)

type ESP32Service struct {
	theaterRepo *repository.TheaterRepository
	roomRepo    *repository.RoomRepository
}

func NewESP32Service(
	theaterRepo *repository.TheaterRepository,
	roomRepo *repository.RoomRepository,
) *ESP32Service {
	return &ESP32Service{
		theaterRepo: theaterRepo,
		roomRepo:    roomRepo,
	}
}

// TelemetryUpdateRequest represents the telemetry data sent by ESP32 devices
// Note: volume_ruangan is NOT included because it's a constant property of the room
type TelemetryUpdateRequest struct {
	Temp          *float64 `json:"temp"`
	Humidity      *int     `json:"humidity"`
	RoomPressure  *float64 `json:"room_pressure"`
	RoomStatus    int      `json:"room_status"`
	LajuAliranAhu int      `json:"laju_aliran_ahu"`
	LogicAhu      int      `json:"logic_ahu"`
	Oxygen        *float64 `json:"oxygen"`
	Nitrous       *float64 `json:"nitrous"`
	Air           *float64 `json:"air"`
	Vacuum        *int     `json:"vacuum"`
	Instrument    *float64 `json:"instrument"`
	Carbon        *float64 `json:"carbon"`
}

// ValidateTelemetryData validates the telemetry data from ESP32
func (s *ESP32Service) ValidateTelemetryData(data *TelemetryUpdateRequest) error {
	// According to the plan, all fields are required
	if data.Temp == nil {
		return errors.New("temp is required")
	}
	if data.Humidity == nil {
		return errors.New("humidity is required")
	}
	if data.RoomPressure == nil {
		return errors.New("room_pressure is required")
	}
	if data.Oxygen == nil {
		return errors.New("oxygen is required")
	}
	if data.Nitrous == nil {
		return errors.New("nitrous is required")
	}
	if data.Air == nil {
		return errors.New("air is required")
	}
	if data.Vacuum == nil {
		return errors.New("vacuum is required")
	}
	if data.Instrument == nil {
		return errors.New("instrument is required")
	}
	if data.Carbon == nil {
		return errors.New("carbon is required")
	}

	// Validate ranges (optional, but good practice)
	if *data.Temp < -50 || *data.Temp > 100 {
		return errors.New("temperature out of valid range (-50 to 100)")
	}
	if *data.Humidity < 0 || *data.Humidity > 100 {
		return errors.New("humidity out of valid range (0 to 100)")
	}
	if *data.RoomPressure < 0 {
		return errors.New("room_pressure cannot be negative")
	}
	if data.RoomStatus < 0 || data.RoomStatus > 1 {
		return errors.New("room_status must be 0 or 1")
	}
	if data.LogicAhu < 0 || data.LogicAhu > 1 {
		return errors.New("logic_ahu must be 0 or 1")
	}

	return nil
}

// UpdateTelemetry updates the telemetry data for a specific room
func (s *ESP32Service) UpdateTelemetry(roomID uint, data *TelemetryUpdateRequest) error {
	// Verify room exists and get room data (we need volume_ruangan from room)
	room, err := s.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return fmt.Errorf("room not found: %w", err)
	}

	// Validate telemetry data
	if err := s.ValidateTelemetryData(data); err != nil {
		return fmt.Errorf("invalid telemetry data: %w", err)
	}

	// Convert request to model
	// Note: VolumeRuangan comes from the room data, not from ESP32
	telemetry := &models.TheaterRawTelemetry{
		RoomID:        &roomID,
		Temp:          data.Temp,
		Humidity:      data.Humidity,
		RoomPressure:  data.RoomPressure,
		RoomStatus:    data.RoomStatus,
		LajuAliranAhu: data.LajuAliranAhu,
		VolumeRuangan: room.VolumeRuangan, // Get from room data
		LogicAhu:      data.LogicAhu,
		Oxygen:        data.Oxygen,
		Nitrous:       data.Nitrous,
		Air:           data.Air,
		Vacuum:        data.Vacuum,
		Instrument:    data.Instrument,
		Carbon:        data.Carbon,
	}

	// Update the raw telemetry table
	// This will trigger the background worker to process the new data
	if err := s.theaterRepo.UpdateRawTelemetryByRoomID(roomID, telemetry); err != nil {
		return fmt.Errorf("failed to update telemetry: %w", err)
	}

	// Log success (optional, could be used for monitoring)
	fmt.Printf("Telemetry updated for room %s (ID: %d)\n", room.RoomCode, roomID)

	return nil
}

// GetRoomTelemetry retrieves the current telemetry data for a room
func (s *ESP32Service) GetRoomTelemetry(roomID uint) (*models.TheaterRawTelemetry, error) {
	telemetry, err := s.theaterRepo.GetRawTelemetryByRoomID(roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get telemetry: %w", err)
	}
	return telemetry, nil
}
