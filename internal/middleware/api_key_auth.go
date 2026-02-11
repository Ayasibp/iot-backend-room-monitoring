package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"iot-backend-room-monitoring/internal/repository"
	"iot-backend-room-monitoring/pkg/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// APIKeyAuthMiddleware validates ESP32 API keys
func APIKeyAuthMiddleware(apiKeyRepo *repository.DeviceAPIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract API key from X-API-Key header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "API key is required in X-API-Key header")
			c.Abort()
			return
		}

		// Trim any whitespace
		apiKey = strings.TrimSpace(apiKey)

		// Extract room_id from URL parameter
		roomIDParam := c.Param("room_id")
		if roomIDParam == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "room_id is required in URL path")
			c.Abort()
			return
		}

		// Parse room_id
		roomID, err := strconv.ParseUint(roomIDParam, 10, 32)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid room_id format")
			c.Abort()
			return
		}

		// Get all API keys for the room and validate
		keys, err := apiKeyRepo.GetAPIKeysByRoomID(uint(roomID))
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to validate API key")
			c.Abort()
			return
		}

		// Try to match the plain key against stored hashes
		valid := false
		for _, key := range keys {
			if !key.IsActive {
				continue
			}

			// Check expiration
			if key.ExpiresAt != nil && key.ExpiresAt.Before(key.CreatedAt) {
				continue
			}

			// Compare the plain key with the hashed key
			err := bcrypt.CompareHashAndPassword([]byte(key.APIKeyHash), []byte(apiKey))
			if err == nil {
				// Key matches!
				valid = true
				break
			}
		}

		if !valid {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired API key")
			c.Abort()
			return
		}

		// Set room_id in context for use by handlers
		c.Set("room_id", uint(roomID))

		// Continue to the next handler
		c.Next()
	}
}
