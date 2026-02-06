package service

import (
	"context"
	"log"
	"time"

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

// processNewTelemetry processes new raw telemetry data and updates live state
func (w *WorkerService) processNewTelemetry() {
	// 1. Get current live state
	liveState, err := w.theaterRepo.GetLiveState("OT-01")
	if err != nil {
		// If live state doesn't exist, skip this cycle
		return
	}

	// 2. Fetch new raw logs since last_processed_raw_id
	rawLogs, err := w.theaterRepo.GetNewRawLogs(liveState.LastProcessedRawID)
	if err != nil {
		log.Printf("Error fetching new raw logs: %v", err)
		return
	}

	// If no new logs, return early
	if len(rawLogs) == 0 {
		return
	}

	// 3. Process each new log
	for _, raw := range rawLogs {
		// METHOD 1: Theoretical ACH = (laju_aliran * 3600) / volume
		if raw.LajuAliranAhu > 0 && raw.VolumeRuangan > 0 {
			liveState.AchTheoretical = float64(raw.LajuAliranAhu*3600) / float64(raw.VolumeRuangan)
		}

		// METHOD 2: Empirical ACH (Edge Detection)
		// 0 -> 1: Start cycle
		if liveState.CurrentLogicAhu == 0 && raw.LogicAhu == 1 {
			liveState.AhuCycleStartTime = &raw.CreatedAt
			log.Printf("ACH cycle started at %v", raw.CreatedAt)
		} else if liveState.CurrentLogicAhu == 1 && raw.LogicAhu == 0 {
			// 1 -> 0: End cycle, calculate
			if liveState.AhuCycleStartTime != nil {
				duration := raw.CreatedAt.Sub(*liveState.AhuCycleStartTime).Seconds()
				if duration > 0 {
					liveState.AchEmpirical = 3600 / duration
					log.Printf("ACH cycle completed - Duration: %.2fs, Empirical ACH: %.2f", duration, liveState.AchEmpirical)
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

		//update medical gases
		liveState.Oxygen = raw.Oxygen
		liveState.Nitrous = raw.Nitrous
		liveState.Air = raw.Air
		liveState.Vacuum = raw.Vacuum
		liveState.Instrument = raw.Instrument
		liveState.Carbon = raw.Carbon

		liveState.CurrentLogicAhu = raw.LogicAhu

		// Update the last processed ID
		liveState.LastProcessedRawID = int(raw.ID)
	}

	// 4. Persist updated state to database
	if err := w.theaterRepo.UpdateLiveState(liveState); err != nil {
		log.Printf("Error updating live state: %v", err)
		return
	}

	log.Printf("Processed %d new telemetry records - Last ID: %d", len(rawLogs), liveState.LastProcessedRawID)
}
