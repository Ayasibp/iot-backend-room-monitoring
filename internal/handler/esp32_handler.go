package handler

import (
	"net/http"
	"time"

	"iot-backend-room-monitoring/internal/service"
	"iot-backend-room-monitoring/pkg/utils"

	"github.com/gin-gonic/gin"
)

type ESP32Handler struct {
	esp32Service *service.ESP32Service
}

func NewESP32Handler(esp32Service *service.ESP32Service) *ESP32Handler {
	return &ESP32Handler{
		esp32Service: esp32Service,
	}
}

// UpdateTelemetry handles telemetry updates from ESP32 devices
// POST /api/v1/esp32/telemetry/:room_id
func (h *ESP32Handler) UpdateTelemetry(c *gin.Context) {
	// Get room_id from context (set by API key middleware)
	roomID, exists := c.Get("room_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Room ID not found in context")
		return
	}

	// Parse request body
	var telemetryData service.TelemetryUpdateRequest
	if err := c.ShouldBindJSON(&telemetryData); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Update telemetry
	if err := h.esp32Service.UpdateTelemetry(roomID.(uint), &telemetryData); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Return success response
	utils.SuccessResponse(c, gin.H{
		"message": "Telemetry updated successfully",
		"data": gin.H{
			"room_id":    roomID,
			"updated_at": time.Now().Format(time.RFC3339),
		},
	})
}

// GetTelemetry retrieves current telemetry data for a room (optional endpoint)
// GET /api/v1/esp32/telemetry/:room_id
func (h *ESP32Handler) GetTelemetry(c *gin.Context) {
	// Get room_id from context (set by API key middleware)
	roomID, exists := c.Get("room_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Room ID not found in context")
		return
	}

	// Get telemetry data
	telemetry, err := h.esp32Service.GetRoomTelemetry(roomID.(uint))
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, telemetry)
}
