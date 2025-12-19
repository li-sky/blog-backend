package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/models"
	"blog-backend/services"
	"blog-backend/utils"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CommentController exposes endpoints for post comments.
type CommentController struct {
	service *services.CommentService
}

func NewCommentController(service *services.CommentService) *CommentController {
	return &CommentController{service: service}
}

type createCommentRequest struct {
	Body string `json:"body" binding:"required"`
}

type visibilityRequest struct {
	Visible bool `json:"visible"`
}

type commentUserResponse struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	EmailSHA256 string `json:"emailSha256,omitempty"`
}

type commentResponse struct {
	ID        uint                 `json:"id"`
	PostID    uint                 `json:"postId"`
	UserID    uint                 `json:"userId"`
	Body      string               `json:"body"`
	Visible   bool                 `json:"visible"`
	User      *commentUserResponse `json:"user,omitempty"`
	UserAgent string               `json:"userAgent,omitempty"`
	Country   string               `json:"country,omitempty"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
}

func emailSHA256(email string) string {
	normalized := strings.TrimSpace(strings.ToLower(email))
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func toCommentResponse(c models.Comment) commentResponse {
	var user *commentUserResponse
	if c.User.ID != 0 {
		user = &commentUserResponse{
			ID:          c.User.ID,
			Username:    c.User.Username,
			EmailSHA256: emailSHA256(c.User.Email),
		}
	}
	return commentResponse{
		ID:        c.ID,
		PostID:    c.PostID,
		UserID:    c.UserID,
		Body:      c.Body,
		Visible:   c.Visible,
		User:      user,
		UserAgent: c.UserAgent,
		Country:   c.Country,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (cc *CommentController) List(c *gin.Context) {
	postID, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit, offset := paginationParams(c)
	comments, total, err := cc.service.ListComments(postID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	responses := make([]commentResponse, 0, len(comments))
	for _, item := range comments {
		responses = append(responses, toCommentResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": responses, "total": total})
}

func (cc *CommentController) Create(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	postID, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	country := utils.CountryByIP(ip)
	comment, err := cc.service.AddComment(user, postID, req.Body, ip, ua, country)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, toCommentResponse(*comment))
}

func (cc *CommentController) Delete(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	commentID, err := parseUintParam(c, "commentID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := cc.service.DeleteComment(user, commentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (cc *CommentController) SetVisibility(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	commentID, err := parseUintParam(c, "commentID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req visibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	if err := cc.service.SetVisibility(user, commentID, req.Visible); err != nil {
		respondServiceError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
