package service

import (
	"context"
	"fmt"
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

	// Create a map of live states by room_id for quick lookup
	// Also maintain room_name map for backward compatibility
	liveStateMapByID := make(map[uint]*models.TheaterLiveState)
	liveStateMapByName := make(map[string]*models.TheaterLiveState)
	for i := range liveStates {
		if liveStates[i].RoomID != nil {
			liveStateMapByID[*liveStates[i].RoomID] = &liveStates[i]
		}
		if liveStates[i].RoomName != "" {
			liveStateMapByName[liveStates[i].RoomName] = &liveStates[i]
		}
	}

	// 3. Process each room's telemetry data
	for _, raw := range rawTelemetry {
		var liveState *models.TheaterLiveState
		var exists bool
		var roomIdentifier string

		// Try to find live state by room_id first (new method)
		if raw.RoomID != nil {
			liveState, exists = liveStateMapByID[*raw.RoomID]
			roomIdentifier = fmt.Sprintf("room_id=%d", *raw.RoomID)
		} else if raw.RoomName != "" {
			// Fallback to room_name (legacy method)
			liveState, exists = liveStateMapByName[raw.RoomName]
			roomIdentifier = fmt.Sprintf("room_name=%s", raw.RoomName)
		} else {
			log.Printf("Warning: Raw telemetry ID %d has no room_id or room_name", raw.ID)
			continue
		}

		if !exists {
			// Create live state if it doesn't exist for this room
			createIdentifier := raw.RoomName
			if createIdentifier == "" && raw.RoomID != nil {
				// If we only have room_id, we can't use the old CreateLiveStateIfNotExists
				log.Printf("Warning: Cannot auto-create live state for room_id %d without room_name", *raw.RoomID)
				continue
			}
			
			if err := w.theaterRepo.CreateLiveStateIfNotExists(createIdentifier); err != nil {
				log.Printf("Error creating live state for %s: %v", roomIdentifier, err)
				continue
			}
			// Fetch the newly created live state
			liveState, err = w.theaterRepo.GetLiveState(createIdentifier)
			if err != nil {
				log.Printf("Error fetching newly created live state for %s: %v", roomIdentifier, err)
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
			log.Printf("Error updating live state for %s: %v", roomIdentifier, err)
			continue
		}

		log.Printf("Processed telemetry for %s - Updated at: %v", roomIdentifier, raw.UpdatedAt)
	}
}

// processRoomTelemetry processes telemetry data for a single room
func (w *WorkerService) processRoomTelemetry(liveState *models.TheaterLiveState, raw *models.TheaterRawTelemetry) {
	// Determine room identifier for logging
	roomIdentifier := "unknown"
	if raw.RoomID != nil {
		roomIdentifier = fmt.Sprintf("room_id=%d", *raw.RoomID)
	} else if raw.RoomName != "" {
		roomIdentifier = raw.RoomName
	}

	// METHOD 1: Theoretical ACH = (laju_aliran * 3600) / volume
	if raw.LajuAliranAhu > 0 && raw.VolumeRuangan > 0 {
		liveState.AchTheoretical = float64(raw.LajuAliranAhu*3600) / float64(raw.VolumeRuangan)
	}

	// METHOD 2: Empirical ACH (Edge Detection)
	// 0 -> 1: Start cycle
	if liveState.CurrentLogicAhu == 0 && raw.LogicAhu == 1 {
		liveState.AhuCycleStartTime = &raw.UpdatedAt
		log.Printf("[%s] ACH cycle started at %v", roomIdentifier, raw.UpdatedAt)
	} else if liveState.CurrentLogicAhu == 1 && raw.LogicAhu == 0 {
		// 1 -> 0: End cycle, calculate
		if liveState.AhuCycleStartTime != nil {
			duration := raw.UpdatedAt.Sub(*liveState.AhuCycleStartTime).Seconds()
			if duration > 0 {
				liveState.AchEmpirical = 3600 / duration
				log.Printf("[%s] ACH cycle completed - Duration: %.2fs, Empirical ACH: %.2f", 
					roomIdentifier, duration, liveState.AchEmpirical)
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
			log.Printf("[%s] Countdown timer expired", roomIdentifier)
		}
	}
}
