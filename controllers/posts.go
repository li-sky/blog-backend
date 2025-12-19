package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/models"
	"blog-backend/services"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// PostController exposes HTTP handlers for blog posts.
type PostController struct {
	service *services.PostService
}

// NewPostController creates a PostController.
func NewPostController(service *services.PostService) *PostController {
	return &PostController{service: service}
}

// Create handles POST /posts.
func (pc *PostController) Create(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var post models.Post
	if err := c.ShouldBindJSON(&post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := pc.service.CreatePost(user, &post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, post)
}

// Update handles PUT /posts/:id.
func (pc *PostController) Update(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var post models.Post
	if err := c.ShouldBindJSON(&post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	post.ID = id
	if err := pc.service.UpdatePost(user, &post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, post)
}

// Publish handles POST /posts/:id/publish.
func (pc *PostController) Publish(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := pc.service.PublishPost(user, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Delete handles DELETE /posts/:id.
func (pc *PostController) Delete(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := pc.service.DeletePost(user, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Get handles GET /posts/:id.
func (pc *PostController) Get(c *gin.Context) {
	id, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	post, err := pc.service.GetPublishedPost(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, post)
}

// List handles GET /posts.
func (pc *PostController) List(c *gin.Context) {
	limit, offset := paginationParams(c)
	sortBy := c.DefaultQuery("sort", "created_at")
	desc := strings.EqualFold(c.DefaultQuery("order", "desc"), "desc")
	posts, total, err := pc.service.ListPublishedPosts(offset, limit, sortBy, desc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := toPostListItems(posts)
	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
	})
}

// ListManage handles GET /posts/manage for editors/admins.
func (pc *PostController) ListManage(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	limit, offset := paginationParams(c)
	sortBy := c.DefaultQuery("sort", "created_at")
	desc := strings.EqualFold(c.DefaultQuery("order", "desc"), "desc")
	posts, total, err := pc.service.ListAllPosts(user, offset, limit, sortBy, desc)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	items := toPostListItems(posts)
	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
	})
}

type postListItem struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	UserID      uint       `json:"userId"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func toPostListItems(posts []models.Post) []postListItem {
	items := make([]postListItem, 0, len(posts))
	for _, p := range posts {
		items = append(items, postListItem{
			ID:          p.ID,
			Title:       p.Title,
			Summary:     p.Summary,
			UserID:      p.UserID,
			Status:      p.Status,
			PublishedAt: p.PublishedAt,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}
	return items
}
