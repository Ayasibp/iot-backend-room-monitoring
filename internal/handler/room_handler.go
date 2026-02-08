package handler

import (
	"net/http"
	"strconv"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/service"
	"iot-backend-room-monitoring/pkg/utils"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	roomService *service.RoomService
}

func NewRoomHandler(roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{
		roomService: roomService,
	}
}

// GetRoomsByHospital retrieves all rooms for a specific hospital
func (h *RoomHandler) GetRoomsByHospital(c *gin.Context) {
	// Parse hospital ID
	hospitalID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid hospital ID")
		return
	}

	// Get user info from context
	userID, _ := c.Get("userID")
	role, _ := c.Get("role")

	rooms, err := h.roomService.GetRoomsByHospitalID(uint(hospitalID), userID.(uint), role.(string))
	if err != nil {
		if err.Error() == "access denied: you don't have permission to access this hospital's rooms" {
			utils.ErrorResponse(c, http.StatusForbidden, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch rooms")
		}
		return
	}

	utils.SuccessResponse(c, gin.H{
		"rooms": rooms,
		"count": len(rooms),
	})
}

// GetRoom retrieves a specific room by ID
func (h *RoomHandler) GetRoom(c *gin.Context) {
	// Parse room ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	// Get user info from context
	userID, _ := c.Get("userID")
	role, _ := c.Get("role")

	room, err := h.roomService.GetRoomByID(uint(id), userID.(uint), role.(string))
	if err != nil {
		if err.Error() == "room not found" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else if err.Error() == "access denied: you don't have permission to access this hospital's rooms" {
			utils.ErrorResponse(c, http.StatusForbidden, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch room")
		}
		return
	}

	utils.SuccessResponse(c, room)
}

// GetAllRooms retrieves all rooms accessible by the user
func (h *RoomHandler) GetAllRooms(c *gin.Context) {
	// Get user info from context
	userID, _ := c.Get("userID")
	role, _ := c.Get("role")

	rooms, err := h.roomService.GetAllRoomsByUser(userID.(uint), role.(string))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch rooms")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"rooms": rooms,
		"count": len(rooms),
	})
}

// CreateRoom creates a new room (admin only)
func (h *RoomHandler) CreateRoom(c *gin.Context) {
	var room models.Room
	if err := c.ShouldBindJSON(&room); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if room.HospitalID == 0 || room.RoomCode == "" || room.RoomName == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "hospital_id, room_code, and room_name are required")
		return
	}

	// Get user ID from context
	userID, _ := c.Get("userID")

	if err := h.roomService.CreateRoom(&room, userID.(uint)); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": "Room created successfully",
		"room":    room,
	})
}

// UpdateRoom updates an existing room (admin only)
func (h *RoomHandler) UpdateRoom(c *gin.Context) {
	// Parse room ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var room models.Room
	if err := c.ShouldBindJSON(&room); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Set the ID from path parameter
	room.ID = uint(id)

	// Get user ID from context
	userID, _ := c.Get("userID")

	if err := h.roomService.UpdateRoom(&room, userID.(uint)); err != nil {
		if err.Error() == "room not found" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		}
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": "Room updated successfully",
		"room":    room,
	})
}

// DeleteRoom soft deletes a room (admin only)
func (h *RoomHandler) DeleteRoom(c *gin.Context) {
	// Parse room ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	// Get user ID from context
	userID, _ := c.Get("userID")

	if err := h.roomService.DeleteRoom(uint(id), userID.(uint)); err != nil {
		if err.Error() == "room not found" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		}
		return
	}

	utils.MessageResponse(c, "Room deleted successfully")
}
