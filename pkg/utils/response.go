package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SuccessResponse sends a standard success JSON response
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// ErrorResponse sends a standard error JSON response
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   message,
	})
}

// MessageResponse sends a simple message response
func MessageResponse(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}
