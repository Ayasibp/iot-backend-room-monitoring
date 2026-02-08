package handler

import (
	"net/http"

	"iot-backend-room-monitoring/internal/service"
	"iot-backend-room-monitoring/pkg/utils"

	"github.com/gin-gonic/gin"
)

type TheaterHandler struct {
	theaterService *service.TheaterService
}

func NewTheaterHandler(theaterService *service.TheaterService) *TheaterHandler {
	return &TheaterHandler{
		theaterService: theaterService,
	}
}

// GetState returns the current live state of the theater room
func (h *TheaterHandler) GetState(c *gin.Context) {
	// Default to OT-01, but can be specified via query parameter
	roomName := c.DefaultQuery("room", "OT-01")

	state, err := h.theaterService.GetLiveState(roomName)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, state)
}

// GetAllStates returns live states for all rooms
func (h *TheaterHandler) GetAllStates(c *gin.Context) {
	states, err := h.theaterService.GetAllLiveStates()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch room states")
		return
	}

	utils.SuccessResponse(c, states)
}

// GetRooms returns a list of all room names
func (h *TheaterHandler) GetRooms(c *gin.Context) {
	rooms, err := h.theaterService.GetAllRooms()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch rooms")
		return
	}

	utils.SuccessResponse(c, map[string]interface{}{
		"rooms": rooms,
		"count": len(rooms),
	})
}

type TimerOperationRequest struct {
	Action string `json:"action" binding:"required,oneof=start stop reset"`
}

// CountdownTimerRequest represents the request body for countdown timer operations
type CountdownTimerRequest struct {
	Action          string `json:"action" binding:"required,oneof=start stop reset"`
	DurationMinutes *int   `json:"duration_minutes"` // Optional, for start action
}

// AdjustTimerRequest represents the request body for adjusting countdown timer
type AdjustTimerRequest struct {
	Minutes int `json:"minutes" binding:"required,oneof=-1 1"`
}

// UpdateTimer handles operation timer control (admin only)
func (h *TheaterHandler) UpdateTimer(c *gin.Context) {
	var req TimerOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request. Action must be 'start', 'stop', or 'reset'")
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get room name from query parameter, default to OT-01
	roomName := c.DefaultQuery("room", "OT-01")

	// Update the timer
	if err := h.theaterService.UpdateOperationTimer(roomName, req.Action, userID.(uint)); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.MessageResponse(c, "Timer operation completed successfully")
}

// UpdateCountdownTimer handles countdown timer control (admin only)
func (h *TheaterHandler) UpdateCountdownTimer(c *gin.Context) {
	var req CountdownTimerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request. Action must be 'start', 'stop', or 'reset'")
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get room name from query parameter, default to OT-01
	roomName := c.DefaultQuery("room", "OT-01")

	// Update the countdown timer
	if err := h.theaterService.UpdateCountdownTimer(roomName, req.Action, req.DurationMinutes, userID.(uint)); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.MessageResponse(c, "Countdown timer operation completed successfully")
}

// AdjustCountdownTimer handles adjusting countdown timer by +/- 1 minute (admin only)
func (h *TheaterHandler) AdjustCountdownTimer(c *gin.Context) {
	var req AdjustTimerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request. Minutes must be -1 or 1")
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get room name from query parameter, default to OT-01
	roomName := c.DefaultQuery("room", "OT-01")

	// Adjust the countdown timer
	if err := h.theaterService.AdjustCountdownTimer(roomName, req.Minutes, userID.(uint)); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.MessageResponse(c, "Countdown timer adjusted successfully")
}
