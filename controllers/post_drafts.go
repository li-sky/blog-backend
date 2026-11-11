package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/models"
	"blog-backend/services"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

type draftInput struct {
	Title   string  `json:"title"`
	Summary string  `json:"summary"`
	Body    string  `json:"body"`
	Version *uint64 `json:"version"`
}

func (pc *PostController) draftError(c *gin.Context, user *models.User, id uint, err error) {
	switch {
	case errors.Is(err, services.ErrDraftConflict):
		draft, _ := pc.service.GetDraft(user, id)
		c.JSON(http.StatusConflict, gin.H{"error": "draft_conflict", "draft": draft})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(404, gin.H{"error": "post not found"})
	case err.Error() == "permission denied":
		c.JSON(403, gin.H{"error": "permission denied"})
	default:
		c.JSON(400, gin.H{"error": err.Error()})
	}
}
func (pc *PostController) GetDraft(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	id, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	draft, err := pc.service.GetDraft(user, id)
	if err != nil {
		pc.draftError(c, user, id, err)
		return
	}
	c.JSON(200, draft)
}
func (pc *PostController) SaveDraft(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	id, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	var input draftInput
	if err = c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid draft"})
		return
	}
	if input.Version == nil {
		c.JSON(409, gin.H{"error": "refresh_required", "message": "Refresh the editor before saving"})
		return
	}
	draft, err := pc.service.SaveDraft(user, id, models.PostDraft{Title: input.Title, Summary: input.Summary, Body: input.Body}, *input.Version)
	if err != nil {
		pc.draftError(c, user, id, err)
		return
	}
	c.JSON(200, draft)
}
func (pc *PostController) DiscardDraft(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	id, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	version, err := strconv.ParseUint(c.Query("version"), 10, 64)
	if err != nil {
		c.JSON(409, gin.H{"error": "refresh_required"})
		return
	}
	draft, err := pc.service.DiscardDraft(user, id, version)
	if err != nil {
		pc.draftError(c, user, id, err)
		return
	}
	c.JSON(200, draft)
}
