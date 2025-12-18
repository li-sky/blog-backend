package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/models"
	"blog-backend/services"
	"blog-backend/utils"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var generateToken = utils.GenerateToken

// AuthController wires HTTP handlers to user-service auth operations.
type AuthController struct {
	userService *services.UserService
}

// NewAuthController creates an AuthController with the provided dependencies.
func NewAuthController(userService *services.UserService) *AuthController {
	return &AuthController{userService: userService}
}

// RegisterRequest captures incoming user registration payloads.
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=12"`
}

// LoginRequest models the login JSON body.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse standardizes login/register responses sent to clients.
type AuthResponse struct {
	Token string  `json:"token"`
	User  userDTO `json:"user"`
}

// userDTO exposes only safe user fields back to the client.
type userDTO struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"createdAt"`
}

const adminRoleName = "admin"

// InitialAdminRegister handles first-time administrator provisioning without prior auth.
func (ac *AuthController) InitialAdminRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Roles: []models.Role{
			{Name: adminRoleName},
		},
	}
	if err := ac.userService.CreateAdminUser(nil, user, true); err != nil {
		respondServiceError(c, err)
		return
	}
	resp, err := buildAuthResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate auth token"})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// Register provisions a new user account and immediately returns an auth token.
func (ac *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}
	if err := ac.userService.Register(user); err != nil {
		respondServiceError(c, err)
		return
	}
	resp, err := buildAuthResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate auth token"})
		return
	}
	c.JSON(http.StatusCreated, resp)

}

// Login authenticates the user and returns a fresh JWT.
func (ac *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	user, err := ac.userService.Authenticate(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	resp, err := buildAuthResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate auth token"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Me returns the authenticated user profile.
func (ac *AuthController) Me(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(user))
}

// buildAuthResponse creates a DTO plus JWT token for an authenticated user.
func buildAuthResponse(user *models.User) (*AuthResponse, error) {
	if user == nil {
		return nil, errors.New("user is required")
	}
	primaryRole := derivePrimaryRole(user)
	token, err := generateToken(user.ID, user.Username, primaryRole)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{
		Token: token,
		User:  toUserDTO(user),
	}, nil
}

func toUserDTO(user *models.User) userDTO {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}
	return userDTO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Roles:     roles,
		CreatedAt: user.CreatedAt,
	}
}

// derivePrimaryRole returns the first role associated with the user, if any.
func derivePrimaryRole(user *models.User) string {
	if user == nil {
		return ""
	}
	if len(user.Roles) == 0 {
		return ""
	}
	return user.Roles[0].Name
}

// respondValidationError emits a 400 with the binding error message.
func respondValidationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

// respondServiceError maps domain errors to HTTP status codes for the client.
func respondServiceError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if err == nil {
		c.JSON(status, gin.H{"error": "unknown error"})
		return
	}
	if strings.Contains(err.Error(), "already") {
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
