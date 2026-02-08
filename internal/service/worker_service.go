package service

import (
	"context"
	"log"
	"time"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/repository"
)

type WorkerService struct {
	theaterRepo *repository.TheaterRepository
}

func NewWorkerService(theaterRepo *repository.TheaterRepository) *WorkerService {
	return &WorkerService{
		theaterRepo: theaterRepo,
	}
}

// Start begins the background worker that processes telemetry data
func (w *WorkerService) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	log.Println("Background worker started - polling every 500ms")

	for {
		select {
		case <-ctx.Done():
			log.Println("Background worker stopped")
			return
		case <-ticker.C:
			w.processNewTelemetry()
		}
	}
}

// processNewTelemetry processes updated raw telemetry data and updates live state for all rooms
func (w *WorkerService) processNewTelemetry() {
	// 1. Get all raw telemetry data (one row per room)
	rawTelemetry, err := w.theaterRepo.GetAllRawTelemetry()
	if err != nil {
		log.Printf("Error fetching raw telemetry: %v", err)
		return
	}

	// If no telemetry data exists, return early
	if len(rawTelemetry) == 0 {
		return
	}

	// 2. Get all live states
	liveStates, err := w.theaterRepo.GetAllLiveStates()
	if err != nil {
		log.Printf("Error fetching live states: %v", err)
		return
	}

	// Create a map of live states by room name for quick lookup
	liveStateMap := make(map[string]*models.TheaterLiveState)
	for i := range liveStates {
		liveStateMap[liveStates[i].RoomName] = &liveStates[i]
	}

	// 3. Process each room's telemetry data
	for _, raw := range rawTelemetry {
		liveState, exists := liveStateMap[raw.RoomName]
		if !exists {
			// Create live state if it doesn't exist for this room
			if err := w.theaterRepo.CreateLiveStateIfNotExists(raw.RoomName); err != nil {
				log.Printf("Error creating live state for room %s: %v", raw.RoomName, err)
				continue
			}
			// Fetch the newly created live state
			liveState, err = w.theaterRepo.GetLiveState(raw.RoomName)
			if err != nil {
				log.Printf("Error fetching newly created live state for room %s: %v", raw.RoomName, err)
				continue
			}
		}

		// Check if the raw data has been updated since last processed
		if liveState.LastProcessedAt != nil && !raw.UpdatedAt.After(*liveState.LastProcessedAt) {
			// No new update for this room, skip
			continue
		}

		// 4. Process the updated telemetry data
		w.processRoomTelemetry(liveState, &raw)

		// 5. Update the last processed timestamp
		liveState.LastProcessedAt = &raw.UpdatedAt

		// 6. Persist updated state to database
		if err := w.theaterRepo.UpdateLiveState(liveState); err != nil {
			log.Printf("Error updating live state for room %s: %v", raw.RoomName, err)
			continue
		}

		log.Printf("Processed telemetry for room %s - Updated at: %v", raw.RoomName, raw.UpdatedAt)
	}
}

// processRoomTelemetry processes telemetry data for a single room
func (w *WorkerService) processRoomTelemetry(liveState *models.TheaterLiveState, raw *models.TheaterRawTelemetry) {
	// METHOD 1: Theoretical ACH = (laju_aliran * 3600) / volume
	if raw.LajuAliranAhu > 0 && raw.VolumeRuangan > 0 {
		liveState.AchTheoretical = float64(raw.LajuAliranAhu*3600) / float64(raw.VolumeRuangan)
	}

	// METHOD 2: Empirical ACH (Edge Detection)
	// 0 -> 1: Start cycle
	if liveState.CurrentLogicAhu == 0 && raw.LogicAhu == 1 {
		liveState.AhuCycleStartTime = &raw.UpdatedAt
		log.Printf("[%s] ACH cycle started at %v", raw.RoomName, raw.UpdatedAt)
	} else if liveState.CurrentLogicAhu == 1 && raw.LogicAhu == 0 {
		// 1 -> 0: End cycle, calculate
		if liveState.AhuCycleStartTime != nil {
			duration := raw.UpdatedAt.Sub(*liveState.AhuCycleStartTime).Seconds()
			if duration > 0 {
				liveState.AchEmpirical = 3600 / duration
				log.Printf("[%s] ACH cycle completed - Duration: %.2fs, Empirical ACH: %.2f", 
					raw.RoomName, duration, liveState.AchEmpirical)
			}
			liveState.AhuCycleStartTime = nil // Reset
		}
	}

	// Update current sensor values from raw data
	if raw.Temp != nil {
		liveState.CurrentTemp = *raw.Temp
	}
	if raw.RoomPressure != nil {
		liveState.CurrentPressure = *raw.RoomPressure
	}

	// Update medical gases
	liveState.Oxygen = raw.Oxygen
	liveState.Nitrous = raw.Nitrous
	liveState.Air = raw.Air
	liveState.Vacuum = raw.Vacuum
	liveState.Instrument = raw.Instrument
	liveState.Carbon = raw.Carbon

	liveState.CurrentLogicAhu = raw.LogicAhu

	// Keep the old LastProcessedRawID for backward compatibility (deprecated)
	liveState.LastProcessedRawID = int(raw.ID)

	// Check countdown timer expiry
	if liveState.CdIsRunning && liveState.CdTargetTime != nil {
		if time.Now().After(*liveState.CdTargetTime) {
			liveState.CdIsRunning = false
			log.Printf("[%s] Countdown timer expired", raw.RoomName)
		}
	}
}
