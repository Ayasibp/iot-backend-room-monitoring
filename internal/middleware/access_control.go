package middleware

import (
	"net/http"
	"strconv"

	"iot-backend-room-monitoring/internal/repository"
	"iot-backend-room-monitoring/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AccessControlMiddleware provides hospital and room access control
type AccessControlMiddleware struct {
	userHospitalRepo *repository.UserHospitalRepository
	roomRepo         *repository.RoomRepository
}

// NewAccessControlMiddleware creates a new access control middleware
func NewAccessControlMiddleware(
	userHospitalRepo *repository.UserHospitalRepository,
	roomRepo *repository.RoomRepository,
) *AccessControlMiddleware {
	return &AccessControlMiddleware{
		userHospitalRepo: userHospitalRepo,
		roomRepo:         roomRepo,
	}
}

// CheckHospitalAccess verifies user has access to the hospital specified in the path
// Expected path parameter: :hospital_id or :id
func (m *AccessControlMiddleware) CheckHospitalAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from context (set by AuthMiddleware)
		userID, exists := c.Get("userID")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
			c.Abort()
			return
		}

		role, exists := c.Get("role")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "User role not found")
			c.Abort()
			return
		}

		// Admin users have access to all hospitals
		if role.(string) == "admin" {
			c.Next()
			return
		}

		// Parse hospital ID from path parameter
		hospitalIDStr := c.Param("hospital_id")
		if hospitalIDStr == "" {
			hospitalIDStr = c.Param("id")
		}

		hospitalID, err := strconv.ParseUint(hospitalIDStr, 10, 32)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid hospital ID")
			c.Abort()
			return
		}

		// Check if user has access to this hospital
		hasAccess, err := m.userHospitalRepo.UserHasAccessToHospital(userID.(uint), uint(hospitalID))
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to verify access")
			c.Abort()
			return
		}

		if !hasAccess {
			utils.ErrorResponse(c, http.StatusForbidden, "Access denied: you don't have permission to access this hospital")
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckRoomAccess verifies user has access to the room specified in the path
// Expected path parameter: :room_id or :id
func (m *AccessControlMiddleware) CheckRoomAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from context (set by AuthMiddleware)
		userID, exists := c.Get("userID")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
			c.Abort()
			return
		}

		role, exists := c.Get("role")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "User role not found")
			c.Abort()
			return
		}

		// Admin users have access to all rooms
		if role.(string) == "admin" {
			c.Next()
			return
		}

		// Parse room ID from path parameter
		roomIDStr := c.Param("room_id")
		if roomIDStr == "" {
			roomIDStr = c.Param("id")
		}

		roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
			c.Abort()
			return
		}

		// Get room to find its hospital
		room, err := m.roomRepo.GetRoomByID(uint(roomID))
		if err != nil {
			utils.ErrorResponse(c, http.StatusNotFound, "Room not found")
			c.Abort()
			return
		}

		// Check if user has access to the room's hospital
		hasAccess, err := m.userHospitalRepo.UserHasAccessToHospital(userID.(uint), room.HospitalID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to verify access")
			c.Abort()
			return
		}

		if !hasAccess {
			utils.ErrorResponse(c, http.StatusForbidden, "Access denied: you don't have permission to access this room")
			c.Abort()
			return
		}

		c.Next()
	}
}
