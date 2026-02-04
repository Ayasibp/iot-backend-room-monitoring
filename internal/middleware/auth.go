package middleware

import (
	"net/http"
	"strings"

	"iot-backend-room-monitoring/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT access token from Authorization header
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid authorization format. Use: Bearer <token>")
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := utils.ValidateAccessToken(token)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		// Inject claims into context
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireAdmin checks if the authenticated user has admin role
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
			c.Abort()
			return
		}

		if role != "admin" {
			utils.ErrorResponse(c, http.StatusForbidden, "Admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
