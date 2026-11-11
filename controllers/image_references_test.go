package controllers

import (
	"blog-backend/models"
	"blog-backend/repositories"
	"blog-backend/services"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func imageReferenceTestApp(t *testing.T) (*gin.Engine, *gorm.DB, *[]string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "images.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Permission{}, &models.Role{}, &models.User{}, &models.Post{}, &models.Image{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	for id := uint(1); id <= 4; id++ {
		require.NoError(t, db.Create(&models.User{Model: gorm.Model{ID: id}, Username: fmt.Sprintf("user%d", id), Email: fmt.Sprintf("user%d@example.com", id)}).Error)
	}
	deleted := []string{}
	backend := &stubStorageBackendForCtrl{deleteFunc: func(name string) error { deleted = append(deleted, name); return nil }}
	controller := NewImageController(services.NewImageService(repositories.NewImageRepo(db), backend))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		var user *models.User
		switch c.GetHeader("X-Test-Auth") {
		case "admin":
			user = createTestUserWithPermissions(1, "upload_image", "edit_any_image", "delete_any_image", "edit_post")
		case "owner":
			user = createTestUserWithPermissions(2, "upload_image")
		case "plain-owner":
			user = createTestUserWithPermissions(2)
		case "other":
			user = createTestUserWithPermissions(3, "upload_image")
		}
		if user != nil {
			c.Set("currentUser", user)
		}
	})
	router.GET("/images/:id/references", controller.GetImageReferences)
	router.DELETE("/images/:id", controller.DeleteImage)
	return router, db, &deleted
}
func imageReferenceRequest(router *gin.Engine, method, path, auth string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("X-Test-Auth", auth)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
func referenceTestImage() *models.Image {
	return &models.Image{Filename: "asset.jpg", OriginalName: "Original.jpg", MimeType: "image/jpeg", URL: "https://skyli.xyz/current/asset.jpg", UserID: 2, StorageBackend: "filesystem", StoragePath: "asset.jpg", WebP600Filename: "asset_w600.webp", WebP600URL: "https://skyli.xyz/current/asset_w600.webp", WebP1200Filename: "asset_w1200.webp", WebP1200URL: "https://skyli.xyz/current/asset_w1200.webp", WebPFullFilename: "asset_full.webp", WebPFullURL: "https://skyli.xyz/current/asset_full.webp"}
}
func TestImageReferencesProtectBothSnapshotsAndDoNotDisclosePrivateDraftTitles(t *testing.T) {
	router, db, deleted := imageReferenceTestApp(t)
	image := referenceTestImage()
	require.NoError(t, db.Create(image).Error)
	published := models.Post{Title: "Public article", Body: "![image](https://skyli.xyz/current/asset.jpg)", Status: "published", UserID: 4, Draft: &models.PostDraft{Title: "SECRET unpublished title", Body: "[download](/images/asset_w600.webp?download=1)"}}
	legacyDraft := models.Post{Title: "SECRET legacy title", Body: "<img src=\"/current/asset_full.webp\">", Status: "draft", UserID: 4}
	require.NoError(t, db.Create(&published).Error)
	require.NoError(t, db.Create(&legacyDraft).Error)
	path := fmt.Sprintf("/images/%d", image.ID)
	require.Equal(t, 401, imageReferenceRequest(router, "GET", path+"/references", "").Code)
	require.Equal(t, 403, imageReferenceRequest(router, "GET", path+"/references", "plain-owner").Code)
	require.Equal(t, 403, imageReferenceRequest(router, "GET", path+"/references", "other").Code)
	admin := imageReferenceRequest(router, "GET", path+"/references", "admin")
	require.Equal(t, 200, admin.Code, admin.Body.String())
	require.Equal(t, "private, no-store", admin.Header().Get("Cache-Control"))
	require.Contains(t, admin.Body.String(), `"total":3`)
	require.Contains(t, admin.Body.String(), `"location":"published"`)
	require.Contains(t, admin.Body.String(), `"location":"draft"`)
	require.Contains(t, admin.Body.String(), "SECRET unpublished title")
	require.NotContains(t, admin.Body.String(), `"body"`)
	owner := imageReferenceRequest(router, "GET", path+"/references", "owner")
	require.Equal(t, 200, owner.Code)
	require.Contains(t, owner.Body.String(), `"total":3`)
	require.NotContains(t, owner.Body.String(), "SECRET")
	for _, auth := range []string{"admin", "owner", "plain-owner"} {
		response := imageReferenceRequest(router, "DELETE", path, auth)
		require.Equal(t, 409, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), "image_in_use")
		require.Empty(t, *deleted)
		if auth != "admin" {
			require.NotContains(t, response.Body.String(), "SECRET")
		}
	}
	require.Equal(t, 403, imageReferenceRequest(router, "DELETE", path, "other").Code)
	require.NoError(t, db.Delete(&published).Error)
	require.NoError(t, db.Delete(&legacyDraft).Error)
	// The server rechecks references after they are removed. Existing owner-delete
	// semantics remain available without requiring a new permission to remove it.
	response := imageReferenceRequest(router, "DELETE", path, "plain-owner")
	require.Equal(t, 204, response.Code, response.Body.String())
	require.Equal(t, []string{"asset.jpg", "asset_w600.webp", "asset_w1200.webp", "asset_full.webp"}, *deleted)
	require.Equal(t, 404, imageReferenceRequest(router, "GET", path+"/references", "admin").Code)
}
func TestImageReferenceLookupFailsClosedAndExternalNamesDoNotBlockDeletion(t *testing.T) {
	router, db, deleted := imageReferenceTestApp(t)
	image := referenceTestImage()
	require.NoError(t, db.Create(image).Error)
	post := models.Post{Title: "Only external links and examples", Body: "![external](https://outside.test/images/asset.jpg)\n\n`![code](/images/asset.jpg)`\n\n![different](/images/prefix-asset.jpg)", Status: "published", UserID: 4}
	require.NoError(t, db.Create(&post).Error)
	path := fmt.Sprintf("/images/%d", image.ID)
	response := imageReferenceRequest(router, "GET", path+"/references", "admin")
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"items":[]`)
	require.Contains(t, response.Body.String(), `"total":0`)
	require.NoError(t, db.Migrator().DropTable(&models.Post{}))
	response = imageReferenceRequest(router, "DELETE", path, "owner")
	require.Equal(t, 500, response.Code)
	require.Empty(t, *deleted, "query errors must not become a permission to delete files")
}

func TestSummaryOnlyCoversPreventDeletionWithoutDisclosingDraftTitles(t *testing.T) {
	router, db, deleted := imageReferenceTestApp(t)
	image := referenceTestImage()
	require.NoError(t, db.Create(image).Error)
	post := models.Post{Title: "Published cover", Summary: "Cover %https://skyli.xyz/current/asset.jpg%", Status: "published", UserID: 4,
		Draft: &models.PostDraft{Title: "SECRET draft cover", Summary: "%/images/asset_w600.webp?cover=1%"}}
	require.NoError(t, db.Create(&post).Error)
	imagePath := fmt.Sprintf("/images/%d", image.ID)
	admin := imageReferenceRequest(router, "GET", imagePath+"/references", "admin")
	require.Equal(t, 200, admin.Code, admin.Body.String())
	var refs services.ImageReferencesResponse
	require.NoError(t, json.Unmarshal(admin.Body.Bytes(), &refs))
	require.Equal(t, 2, refs.Total)
	require.Len(t, refs.Items, 2)
	require.Equal(t, "published", refs.Items[0].Location)
	require.Equal(t, "draft", refs.Items[1].Location)
	require.Equal(t, "SECRET draft cover", refs.Items[1].Title)
	owner := imageReferenceRequest(router, "GET", imagePath+"/references", "owner")
	require.Equal(t, 200, owner.Code)
	require.NotContains(t, owner.Body.String(), "SECRET")
	require.NoError(t, json.Unmarshal(owner.Body.Bytes(), &refs))
	require.Equal(t, 2, refs.Total)
	require.Len(t, refs.Items, 1)
	response := imageReferenceRequest(router, "DELETE", imagePath, "owner")
	require.Equal(t, 409, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "image_in_use")
	require.NotContains(t, response.Body.String(), "SECRET")
	require.Empty(t, *deleted)
	// Repeating the same cover in each body's snapshot must not duplicate locations.
	post.Body = "![image](/images/asset.jpg)"
	post.Draft.Body = "![image](/images/asset_w600.webp)"
	require.NoError(t, db.Save(&post).Error)
	admin = imageReferenceRequest(router, "GET", imagePath+"/references", "admin")
	require.NoError(t, json.Unmarshal(admin.Body.Bytes(), &refs))
	require.Equal(t, 2, refs.Total)
	require.Len(t, refs.Items, 2)
	// A private draft cover alone still blocks deletion without exposing its title.
	post.Summary, post.Body, post.Draft.Body = "", "", ""
	require.NoError(t, db.Save(&post).Error)
	response = imageReferenceRequest(router, "DELETE", imagePath, "owner")
	require.Equal(t, 409, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"total":1`)
	require.Contains(t, response.Body.String(), `"items":[]`)
	require.NotContains(t, response.Body.String(), "SECRET")
	require.Empty(t, *deleted)
}
