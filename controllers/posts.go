package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/models"
	"blog-backend/services"
	"fmt"
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
	var input struct {
		Title           string  `json:"title"`
		Summary         string  `json:"summary"`
		Body            string  `json:"body"`
		Status          string  `json:"status"`
		ClientRequestID *string `json:"clientRequestId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	post := models.Post{Title: input.Title, Summary: input.Summary, Body: input.Body, Status: input.Status, ClientRequestID: input.ClientRequestID}
	if err := pc.service.CreatePost(user, &post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response := toPostDetailResponse(&post)
	response.DraftVersion = &post.DraftVersion
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusCreated, response)
}

// Update handles PUT /posts/:id.
func (pc *PostController) Update(c *gin.Context) { pc.SaveDraft(c) }

// Publish handles POST /posts/:id/publish.
func (pc *PostController) Publish(c *gin.Context) {
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
	var input struct {
		Version *uint64 `json:"version"`
	}
	if err = c.ShouldBindJSON(&input); err != nil || input.Version == nil {
		c.JSON(409, gin.H{"error": "refresh_required", "message": "Refresh the editor before publishing"})
		return
	}
	draft, err := pc.service.PublishDraft(user, id, *input.Version)
	if err != nil {
		pc.draftError(c, user, id, err)
		return
	}
	c.JSON(200, draft)
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
	c.Header("Cache-Control", "private, no-store")
	id, err := parseUintParam(c, "postID")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, _ := middlewares.CurrentUser(c)
	if user != nil {
		fmt.Printf("PostController.Get: Requesting post %d by user %s (ID: %d)\n", id, user.Username, user.ID)
	} else {
		fmt.Printf("PostController.Get: Requesting post %d by anonymous\n", id)
	}

	post, err := pc.service.GetPostForUser(user, id)
	if err != nil {
		fmt.Printf("PostController.Get: Error getting post: %v\n", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPostDetailResponse(post))
}

// List handles GET /posts.
func (pc *PostController) List(c *gin.Context) {
	limit, offset := paginationParams(c)
	sortBy := c.DefaultQuery("sort", "published_at")
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
	c.Header("Cache-Control", "private, no-store")
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
	for index, post := range posts {
		items[index].HasUnpublishedChanges = post.Status != "published" || (post.Draft != nil && (post.Draft.Title != post.Title || post.Draft.Summary != post.Summary || post.Draft.Body != post.Body))
		if post.Draft != nil {
			items[index].Title = post.Draft.Title
			items[index].Summary = post.Draft.Summary
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
	})
}

type postListItem struct {
	ID                    uint       `json:"id"`
	Title                 string     `json:"title"`
	Summary               string     `json:"summary"`
	UserID                uint       `json:"userId"`
	Status                string     `json:"status"`
	PublishedAt           *time.Time `json:"publishedAt,omitempty"`
	PublishedUpdatedAt    *time.Time `json:"publishedUpdatedAt,omitempty"`
	PublicationTimeKnown  bool       `json:"publicationTimeKnown"`
	HasUnpublishedChanges bool       `json:"hasUnpublishedChanges,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
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
			PublishedAt: p.PublishedAt, PublishedUpdatedAt: p.PublishedUpdatedAt, PublicationTimeKnown: p.PublicationTimeKnown,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return items
}

type postDetailResponse struct {
	ID                   uint          `json:"id"`
	Title                string        `json:"title"`
	Summary              string        `json:"summary"`
	Body                 string        `json:"body"`
	UserID               uint          `json:"userId"`
	User                 *userResponse `json:"user,omitempty"`
	Status               string        `json:"status"`
	PublishedAt          *time.Time    `json:"publishedAt,omitempty"`
	PublishedUpdatedAt   *time.Time    `json:"publishedUpdatedAt,omitempty"`
	PublicationTimeKnown bool          `json:"publicationTimeKnown"`
	DraftVersion         *uint64       `json:"draftVersion,omitempty"`
	CreatedAt            time.Time     `json:"createdAt"`
	UpdatedAt            time.Time     `json:"updatedAt"`
}

func toPostDetailResponse(post *models.Post) postDetailResponse {
	if post == nil {
		return postDetailResponse{}
	}
	return postDetailResponse{
		ID:          post.ID,
		Title:       post.Title,
		Summary:     post.Summary,
		Body:        post.Body,
		UserID:      post.UserID,
		User:        newUserResponse(post.User),
		Status:      post.Status,
		PublishedAt: post.PublishedAt, PublishedUpdatedAt: post.PublishedUpdatedAt, PublicationTimeKnown: post.PublicationTimeKnown,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
	}
}
