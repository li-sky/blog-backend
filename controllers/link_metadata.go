package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/services"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var metadataRequests = make(chan struct{}, 4)

// Metadata is author-only and never forwards user cookies or authorization.
func LinkMetadata(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	allowed := false
	for _, role := range user.Roles {
		for _, permission := range role.Permissions {
			if permission.Name == "upload_image" {
				allowed = true
			}
		}
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied: upload_image required"})
		return
	}
	select {
	case metadataRequests <- struct{}{}:
		defer func() { <-metadataRequests }()
	default:
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "链接信息读取繁忙，请稍后重试"})
		return
	}
	c.Header("Cache-Control", "no-store")
	raw := c.Query("url")
	if len(raw) == 0 || len(raw) > 4096 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a URL of at most 4096 characters is required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	metadata, err := services.FetchLinkMetadata(ctx, raw)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "无法读取网站信息，请手动填写标题或图标"})
		return
	}
	c.JSON(http.StatusOK, metadata)
}
