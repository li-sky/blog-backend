package router

import (
	"blog-backend/controllers"
	"blog-backend/middlewares"
	"blog-backend/services"

	"github.com/gin-gonic/gin"
)

// SetupRouter wires all HTTP routes and returns a configured Gin engine.
func SetupRouter(
	userService *services.UserService,
	roleService *services.RoleService,
	postService *services.PostService,
	commentService *services.CommentService,
	settingService *services.SettingService,
	imageService *services.ImageService,
) *gin.Engine {
	r := gin.Default()
	r.Use(middlewares.CORSMiddleware())

	authController := controllers.NewAuthController(userService)
	userController := controllers.NewUserController(userService)
	roleController := controllers.NewRoleController(roleService)
	postController := controllers.NewPostController(postService)
	commentController := controllers.NewCommentController(commentService)
	settingController := controllers.NewSettingController(settingService)
	feedController := controllers.NewFeedController(postService, settingService)
	imageController := controllers.NewImageController(imageService)

	api := r.Group("/api/v1")

	auth := api.Group("/auth")
	auth.POST("/register", authController.Register)
	auth.POST("/login", authController.Login)
	auth.POST("/admin/init", authController.InitialAdminRegister)
	authProtected := auth.Group("")
	authProtected.Use(middlewares.AuthMiddleware(userService))
	authProtected.GET("/me", authController.Me)

	users := api.Group("/users")
	users.Use(middlewares.AuthMiddleware(userService))
	users.GET("", userController.List)
	users.GET("/:id", userController.Get)
	users.PUT("/:id", userController.Update)
	users.DELETE("/:id", userController.Delete)
	users.PUT("/:id/roles", userController.SetRoles)
	users.POST("/admin", userController.CreateAdmin)

	roles := api.Group("/roles")
	roles.Use(middlewares.AuthMiddleware(userService))
	roles.GET("", roleController.List)
	roles.GET("/:id", roleController.Get)
	roles.POST("", roleController.Create)
	roles.PUT("/:id", roleController.Update)
	roles.DELETE("/:id", roleController.Delete)

	posts := api.Group("/posts")
	posts.GET("", postController.List)
	posts.GET("/:postID", postController.Get)

	protectedPosts := posts.Group("")
	protectedPosts.Use(middlewares.AuthMiddleware(userService))
	protectedPosts.POST("", postController.Create)
	protectedPosts.GET("/manage", postController.ListManage)
	protectedPosts.PUT("/:postID", postController.Update)
	protectedPosts.POST("/:postID/publish", postController.Publish)
	protectedPosts.DELETE("/:postID", postController.Delete)

	comments := posts.Group("/:postID/comments")
	comments.GET("", commentController.List)
	authedComments := comments.Group("")
	authedComments.Use(middlewares.AuthMiddleware(userService))
	authedComments.POST("", commentController.Create)
	authedComments.DELETE("/:commentID", commentController.Delete)
	authedComments.PUT("/:commentID/visibility", commentController.SetVisibility)

	settings := api.Group("/settings")
	settings.GET("", settingController.List)
	settingsProtected := settings.Group("")
	settingsProtected.Use(middlewares.AuthMiddleware(userService))
	settingsProtected.PUT("/:key", settingController.Update)

	images := api.Group("/images")
	images.Use(middlewares.AuthMiddleware(userService))
	images.POST("/upload", imageController.UploadImage)
	images.GET("", imageController.ListImages)
	images.GET("/:id", imageController.GetImage)
	images.PUT("/:id", imageController.UpdateImage)
	images.DELETE("/:id", imageController.DeleteImage)

	feed := r.Group("/feed")
	feed.GET("/atom", feedController.AtomFeed)
	feed.GET("/rss", feedController.RSSFeed)

	return r
}
