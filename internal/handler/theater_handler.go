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
	// Default to OT-01, but could be parameterized
	roomName := c.DefaultQuery("room", "OT-01")

	state, err := h.theaterService.GetLiveState(roomName)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, state)
}

type TimerOperationRequest struct {
	Action string `json:"action" binding:"required,oneof=start stop reset"`
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

	// Default to OT-01
	roomName := "OT-01"

	// Update the timer
	if err := h.theaterService.UpdateOperationTimer(roomName, req.Action, userID.(uint)); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.MessageResponse(c, "Timer operation completed successfully")
}
