package handler

import (
	"net/http"
	"time"

	"iot-backend-room-monitoring/internal/service"
	"iot-backend-room-monitoring/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"omitempty,oneof=admin user"`
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Authenticate user
	response, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	// Set refresh token as HttpOnly cookie
	c.SetCookie(
		"refresh_token",               // name
		response.RefreshToken,         // value
		int(7*24*time.Hour.Seconds()), // maxAge in seconds (7 days)
		"/",                           // path
		"",                            // domain (empty means current domain)
		false,                         // secure (set to true in production with HTTPS)
		true,                          // httpOnly
	)

	// Return access token and user info in JSON
	utils.SuccessResponse(c, gin.H{
		"access_token": response.AccessToken,
		"user":         response.User,
	})
}

// Refresh generates a new access token from refresh token
func (h *AuthHandler) Refresh(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Refresh token not found")
		return
	}

	// Generate new access token
	accessToken, err := h.authService.RefreshAccessToken(refreshToken)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"access_token": accessToken,
	})
}

// Logout revokes the refresh token
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		// If no cookie, just clear it and return success
		c.SetCookie("refresh_token", "", -1, "/", "", false, true)
		utils.MessageResponse(c, "Logged out successfully")
		return
	}

	// Revoke the refresh token
	if err := h.authService.Logout(refreshToken); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to logout")
		return
	}

	// Clear the cookie
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	utils.MessageResponse(c, "Logged out successfully")
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set default role if not specified
	if req.Role == "" {
		req.Role = "user"
	}

	// Register user
	response, err := h.authService.Register(req.Username, req.Password, req.Role)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Set refresh token as HttpOnly cookie
	c.SetCookie(
		"refresh_token",               // name
		response.RefreshToken,         // value
		int(7*24*time.Hour.Seconds()), // maxAge in seconds (7 days)
		"/",                           // path
		"",                            // domain (empty means current domain)
		false,                         // secure (set to true in production with HTTPS)
		true,                          // httpOnly
	)

	// Return access token and user info in JSON
	utils.SuccessResponse(c, gin.H{
		"access_token": response.AccessToken,
		"user":         response.User,
	})
}
