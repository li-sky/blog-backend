package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RoleController handles role CRUD endpoints.
type RoleController struct {
	service *services.RoleService
}

func NewRoleController(service *services.RoleService) *RoleController {
	return &RoleController{service: service}
}

type permissionPayload struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type roleCreateRequest struct {
	Name        string              `json:"name" binding:"required"`
	Permissions []permissionPayload `json:"permissions"`
}

type roleUpdateRequest struct {
	Name        string              `json:"name"`
	Permissions []permissionPayload `json:"permissions"`
}

func (rc *RoleController) List(c *gin.Context) {
	operator, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	limit, offset := paginationParams(c)
	roles, total, err := rc.service.ListRoles(operator, offset, limit)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": roles, "total": total})
}

func (rc *RoleController) Get(c *gin.Context) {
	operator, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role, err := rc.service.GetRole(operator, id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, role)
}

func (rc *RoleController) Create(c *gin.Context) {
	operator, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req roleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	role, err := rc.service.CreateRole(operator, req.Name, toPermissionInputs(req.Permissions))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, role)
}

func (rc *RoleController) Update(c *gin.Context) {
	operator, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req roleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	role, err := rc.service.UpdateRole(operator, id, req.Name, toPermissionInputs(req.Permissions))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, role)
}

func (rc *RoleController) Delete(c *gin.Context) {
	operator, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := rc.service.DeleteRole(operator, id); err != nil {
		respondServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toPermissionInputs(payloads []permissionPayload) []services.PermissionInput {
	inputs := make([]services.PermissionInput, 0, len(payloads))
	for _, p := range payloads {
		inputs = append(inputs, services.PermissionInput{Name: p.Name, Description: p.Description})
	}
	return inputs
}
