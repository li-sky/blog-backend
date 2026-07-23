package controllers

import (
	"blog-backend/middlewares"
	"blog-backend/services"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ImageController struct {
	imageService *services.ImageService
}

func NewImageController(imageService *services.ImageService) *ImageController {
	return &ImageController{
		imageService: imageService,
	}
}

// UploadImage 上传图片
func (ctrl *ImageController) UploadImage(c *gin.Context) {
	// 获取当前登录用户
	currentUser, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 获取上传的文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// 获取可选的描述文本
	alt := c.PostForm("alt")
	webp600, _ := c.FormFile("webp600")
	webp1200, _ := c.FormFile("webp1200")
	webpFull, _ := c.FormFile("webpFull")

	// 获取客户端 IP
	ipAddress := c.ClientIP()

	// 调用服务层上传图片
	image, err := ctrl.imageService.UploadImageWithVariants(
		currentUser,
		fileHeader,
		alt,
		ipAddress,
		services.ImageVariantFiles{WebP600: webp600, WebP1200: webp1200, WebPFull: webpFull},
	)
	if err != nil {
		// 检查权限错误
		if strings.Contains(err.Error(), "permission denied") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, services.ErrInvalidImageFormat) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image format"})
			return
		}
		if errors.Is(err, services.ErrImageTooLarge) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image size exceeds 50MB limit"})
			return
		}
		if errors.Is(err, services.ErrInvalidVariant) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 记录详细错误用于调试
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload image", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, image)
}

// UpdateImageVariants upgrades a legacy image with browser-generated WebP files.
func (ctrl *ImageController) UpdateImageVariants(c *gin.Context) {
	currentUser, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image id"})
		return
	}

	webp600, _ := c.FormFile("webp600")
	webp1200, _ := c.FormFile("webp1200")
	webpFull, _ := c.FormFile("webpFull")
	if webp600 == nil && webp1200 == nil && webpFull == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one image variant is required"})
		return
	}

	image, err := ctrl.imageService.UpdateImageVariants(
		currentUser,
		uint(id),
		services.ImageVariantFiles{WebP600: webp600, WebP1200: webp1200, WebPFull: webpFull},
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrImageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		case errors.Is(err, services.ErrInvalidVariant):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case strings.Contains(err.Error(), "permission denied"):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update image variants", "details": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, image)
}

// ListImages 获取图片列表
func (ctrl *ImageController) ListImages(c *gin.Context) {
	// 获取分页参数
	limit, offset := paginationParams(c)

	// 调用服务层
	images, total, err := ctrl.imageService.ListImages(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list images"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": images,
		"total": total,
	})
}

// GetImage 获取图片详情
func (ctrl *ImageController) GetImage(c *gin.Context) {
	// 获取图片 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image id"})
		return
	}

	// 调用服务层
	image, err := ctrl.imageService.GetImage(uint(id))
	if err != nil {
		if errors.Is(err, services.ErrImageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get image"})
		return
	}

	c.JSON(http.StatusOK, image)
}

// UpdateImage 更新图片信息
func (ctrl *ImageController) UpdateImage(c *gin.Context) {
	// 获取图片 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image id"})
		return
	}

	// 获取当前登录用户
	currentUser, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Alt string `json:"alt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 调用服务层更新图片
	image, err := ctrl.imageService.UpdateImage(currentUser, uint(id), req.Alt)
	if err != nil {
		if errors.Is(err, services.ErrImageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update image"})
		return
	}

	c.JSON(http.StatusOK, image)
}

// DeleteImage 删除图片
func (ctrl *ImageController) DeleteImage(c *gin.Context) {
	// 获取图片 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image id"})
		return
	}

	// 获取当前登录用户
	currentUser, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 调用服务层
	err = ctrl.imageService.DeleteImage(currentUser, uint(id))
	if err != nil {
		if errors.Is(err, services.ErrImageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete image"})
		return
	}

	c.Status(http.StatusNoContent)
}
