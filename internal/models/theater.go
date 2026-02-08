package models

import "time"

// TheaterRawTelemetry represents the theater_raw_telemetry table
// Hardware updates a single row per room with raw telemetry data
type TheaterRawTelemetry struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RoomID    *uint     `gorm:"index" json:"room_id"`                                  // New: FK to rooms table
	RoomName  string    `gorm:"size:50;uniqueIndex;default:'OT-01'" json:"room_name"` // Kept for backward compatibility
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`

	// Sensor inputs from hardware
	Temp         *float64 `json:"temp"`
	Humidity     *int     `json:"humidity"`
	RoomPressure *float64 `gorm:"column:room_pressure" json:"room_pressure"`
	RoomStatus   int      `gorm:"default:0" json:"room_status"` // 0=Off, 1=On

	// ACH Calculation inputs
	LajuAliranAhu int `gorm:"column:laju_aliran_ahu;default:0" json:"laju_aliran_ahu"` // Flow rate
	VolumeRuangan int `gorm:"column:volume_ruangan;default:0" json:"volume_ruangan"`   // Room volume
	LogicAhu      int `gorm:"column:logic_ahu;default:0" json:"logic_ahu"`             // Trigger (0 or 1)

	// Medical gases
	Oxygen     *float64 `json:"oxygen"`
	Nitrous    *float64 `json:"nitrous"`
	Air        *float64 `json:"air"`
	Vacuum     *int     `json:"vacuum"`
	Instrument *float64 `json:"instrument"`
	Carbon     *float64 `json:"carbon"`

	// Relationships
	Room Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

// TableName specifies the table name for TheaterRawTelemetry model
func (TheaterRawTelemetry) TableName() string {
	return "theater_raw_telemetry"
}

// TheaterLiveState represents the theater_live_state table
// The background worker updates this single row with calculated results
type TheaterLiveState struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	RoomID   *uint  `gorm:"index" json:"room_id"`                                     // New: FK to rooms table
	RoomName string `gorm:"size:50;uniqueIndex;default:'OT-01'" json:"room_name"` // Kept for backward compatibility

	// A. Calculated results from worker
	AchTheoretical float64 `gorm:"column:ach_theoretical;default:0.0" json:"ach_theoretical"` // Method 1
	AchEmpirical   float64 `gorm:"column:ach_empirical;default:0.0" json:"ach_empirical"`     // Method 2

	// B. Latest sensor values (copied from raw)
	CurrentTemp     float64 `gorm:"column:current_temp;default:0.0" json:"current_temp"`
	CurrentPressure float64 `gorm:"column:current_pressure;default:0.0" json:"current_pressure"`
	CurrentLogicAhu int     `gorm:"column:current_logic_ahu;default:0" json:"current_logic_ahu"`

	// C. Stopwatch logic (admin controlled)
	OpStartTime          *time.Time `gorm:"column:op_start_time" json:"op_start_time"`
	OpAccumulatedSeconds int        `gorm:"column:op_accumulated_seconds;default:0" json:"op_accumulated_seconds"`
	OpIsRunning          bool       `gorm:"column:op_is_running;default:false" json:"op_is_running"`

	// D. Countdown logic (admin controlled)
	CdTargetTime      *time.Time `gorm:"column:cd_target_time" json:"cd_target_time"`
	CdDurationSeconds int        `gorm:"column:cd_duration_seconds;default:3600" json:"cd_duration_seconds"`
	CdIsRunning       bool       `gorm:"column:cd_is_running;default:false" json:"cd_is_running"`

	// E. Internal worker state (for Method 2 timing)
	AhuCycleStartTime  *time.Time `gorm:"column:ahu_cycle_start_time" json:"ahu_cycle_start_time"`
	LastProcessedRawID int        `gorm:"column:last_processed_raw_id;default:0" json:"last_processed_raw_id"` // Deprecated
	LastProcessedAt    *time.Time `gorm:"column:last_processed_at" json:"last_processed_at"`                    // Track last processed timestamp

	// F. Medical gases
	Oxygen     *float64 `json:"oxygen"`
	Nitrous    *float64 `json:"nitrous"`
	Air        *float64 `json:"air"`
	Vacuum     *int     `json:"vacuum"`
	Instrument *float64 `json:"instrument"`
	Carbon     *float64 `json:"carbon"`

	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	Room Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

// TableName specifies the table name for TheaterLiveState model
func (TheaterLiveState) TableName() string {
	return "theater_live_state"
}
