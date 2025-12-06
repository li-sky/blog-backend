package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SettingController exposes read/write endpoints for application settings.
type SettingController struct {
	service *services.SettingService
}

func NewSettingController(service *services.SettingService) *SettingController {
	return &SettingController{service: service}
}

func (sc *SettingController) List(c *gin.Context) {
	settings, err := sc.service.ListSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": settings})
}

type updateSettingRequest struct {
	Value       string `json:"value" binding:"required"`
	Description string `json:"description"`
}

func (sc *SettingController) Update(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	key := strings.TrimSpace(c.Param("key"))
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "setting key is required"})
		return
	}
	var req updateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	setting, err := sc.service.UpdateSetting(user, key, req.Value, req.Description)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, setting)
}
