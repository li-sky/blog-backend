package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/models"
	"blog-backend/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// UserController exposes management endpoints for user accounts.
type UserController struct {
	service *services.UserService
}

func NewUserController(service *services.UserService) *UserController {
	return &UserController{service: service}
}

func (uc *UserController) List(c *gin.Context) {
	operator, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	limit, offset := paginationParams(c)
	users, total, err := uc.service.ListUsers(operator, offset, limit)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	items := make([]userDTO, len(users))
	for i := range users {
		items[i] = toUserDTO(&users[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (uc *UserController) Get(c *gin.Context) {
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
	user, err := uc.service.GetUser(operator, id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(user))
}

type updateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password"`
}

func (uc *UserController) Update(c *gin.Context) {
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
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := &models.User{Username: req.Username, Email: req.Email, Password: req.Password}
	user.ID = id
	if err := uc.service.UpdateUser(operator, user); err != nil {
		respondServiceError(c, err)
		return
	}
	updated, _ := uc.service.GetByID(id)
	c.JSON(http.StatusOK, toUserDTO(updated))
}

type setRolesRequest struct {
	RoleIDs []uint `json:"roleIds"`
}

func (uc *UserController) SetRoles(c *gin.Context) {
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
	var req setRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := uc.service.SetUserRoles(operator, id, req.RoleIDs)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(user))
}

func (uc *UserController) Delete(c *gin.Context) {
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
	if err := uc.service.DeleteUser(operator, id); err != nil {
		respondServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (uc *UserController) CreateAdmin(c *gin.Context) {
	operator, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	user := &models.User{Username: req.Username, Email: req.Email, Password: req.Password}
	if err := uc.service.CreateAdminUser(operator, user, false); err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserDTO(user))
}
