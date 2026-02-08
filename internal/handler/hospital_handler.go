package handler

import (
	"net/http"
	"strconv"

	"iot-backend-room-monitoring/internal/models"
	"iot-backend-room-monitoring/internal/service"
	"iot-backend-room-monitoring/pkg/utils"

	"github.com/gin-gonic/gin"
)

type HospitalHandler struct {
	hospitalService *service.HospitalService
}

func NewHospitalHandler(hospitalService *service.HospitalService) *HospitalHandler {
	return &HospitalHandler{
		hospitalService: hospitalService,
	}
}

// GetAllHospitals retrieves all hospitals accessible by the user
// Admin users see all hospitals, regular users see only assigned hospitals
func (h *HospitalHandler) GetAllHospitals(c *gin.Context) {
	// Get user info from context (set by auth middleware)
	userID, _ := c.Get("userID")
	role, _ := c.Get("role")

	hospitals, err := h.hospitalService.GetAllHospitals(userID.(uint), role.(string))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch hospitals")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"hospitals": hospitals,
		"count":     len(hospitals),
	})
}

// GetHospital retrieves a specific hospital by ID
func (h *HospitalHandler) GetHospital(c *gin.Context) {
	// Parse hospital ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid hospital ID")
		return
	}

	// Get user info from context
	userID, _ := c.Get("userID")
	role, _ := c.Get("role")

	hospital, err := h.hospitalService.GetHospitalByID(uint(id), userID.(uint), role.(string))
	if err != nil {
		if err.Error() == "hospital not found" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else if err.Error() == "access denied: you don't have permission to view this hospital" {
			utils.ErrorResponse(c, http.StatusForbidden, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch hospital")
		}
		return
	}

	utils.SuccessResponse(c, hospital)
}

// CreateHospital creates a new hospital (admin only)
func (h *HospitalHandler) CreateHospital(c *gin.Context) {
	var hospital models.Hospital
	if err := c.ShouldBindJSON(&hospital); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if hospital.Code == "" || hospital.Name == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Code and name are required")
		return
	}

	// Get user ID from context
	userID, _ := c.Get("userID")

	if err := h.hospitalService.CreateHospital(&hospital, userID.(uint)); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message":  "Hospital created successfully",
		"hospital": hospital,
	})
}

// UpdateHospital updates an existing hospital (admin only)
func (h *HospitalHandler) UpdateHospital(c *gin.Context) {
	// Parse hospital ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid hospital ID")
		return
	}

	var hospital models.Hospital
	if err := c.ShouldBindJSON(&hospital); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Set the ID from path parameter
	hospital.ID = uint(id)

	// Get user ID from context
	userID, _ := c.Get("userID")

	if err := h.hospitalService.UpdateHospital(&hospital, userID.(uint)); err != nil {
		if err.Error() == "hospital not found" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		}
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message":  "Hospital updated successfully",
		"hospital": hospital,
	})
}

// DeleteHospital soft deletes a hospital (admin only)
func (h *HospitalHandler) DeleteHospital(c *gin.Context) {
	// Parse hospital ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid hospital ID")
		return
	}

	// Get user ID from context
	userID, _ := c.Get("userID")

	if err := h.hospitalService.DeleteHospital(uint(id), userID.(uint)); err != nil {
		if err.Error() == "hospital not found" {
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		}
		return
	}

	utils.MessageResponse(c, "Hospital deleted successfully")
}
