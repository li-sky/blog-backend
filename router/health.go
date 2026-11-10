package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// registerHealthRoutes reports readiness without disclosing database errors or credentials.
func registerHealthRoutes(r *gin.Engine, pingDatabase func(context.Context) error) {
	handler := func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if pingDatabase == nil || pingDatabase(ctx) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
	r.GET("/health", handler)
	r.GET("/api/v1/health", handler)
}
