package controllers

import (
	"blog-backend/models"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLinkMetadataAuthorPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		user   *models.User
		status int
	}{
		{"anonymous", nil, 401},
		{"reader", &models.User{}, 403},
		{"author invalid URL", &models.User{Roles: []models.Role{{Permissions: []models.Permission{{Name: "upload_image"}}}}}, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/metadata", func(c *gin.Context) {
				if tc.user != nil {
					c.Set("currentUser", tc.user)
				}
				LinkMetadata(c)
			})
			response := httptest.NewRecorder()
			r.ServeHTTP(response, httptest.NewRequest("GET", "/metadata", nil))
			if response.Code != tc.status {
				t.Fatalf("got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestLinkMetadataRejectsPrivateURLsAndSaturation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/metadata", func(c *gin.Context) {
		c.Set("currentUser", &models.User{Roles: []models.Role{{Permissions: []models.Permission{{Name: "upload_image"}}}}})
		LinkMetadata(c)
	})
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest("GET", "/metadata?url=http%3A%2F%2F127.0.0.1%2F", nil))
	if response.Code != 422 {
		t.Fatalf("private destination returned %d", response.Code)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("metadata responses must not cache author data")
	}
	for i := 0; i < cap(metadataRequests); i++ {
		metadataRequests <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(metadataRequests); i++ {
			<-metadataRequests
		}
	}()
	response = httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest("GET", "/metadata?url=https%3A%2F%2Fexample.com", nil))
	if response.Code != 429 {
		t.Fatalf("saturated limiter returned %d", response.Code)
	}
}
