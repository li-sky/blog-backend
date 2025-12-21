package main

import (
	"log"

	"blog-backend/config"
	"blog-backend/models"
	"blog-backend/repositories"
	"blog-backend/router"
	"blog-backend/services"
	"blog-backend/utils/storage"
)

func main() {
	db := config.InitDB()
	if db == nil {
		log.Fatal("failed to initialize database")
	}

	if err := db.AutoMigrate(
		&models.Permission{},
		&models.Role{},
		&models.User{},
		&models.Post{},
		&models.Comment{},
		&models.Setting{},
		&models.Image{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	userRepo := repositories.NewUserRepository(db)
	roleRepo := repositories.NewRoleRepository(db)
	postRepo := repositories.NewPostRepository(db)
	commentRepo := repositories.NewCommentRepository(db)
	settingRepo := repositories.NewSettingRepository(db)
	imageRepo := repositories.NewImageRepo(db)

	userService := services.NewUserService(userRepo, roleRepo)
	roleService := services.NewRoleService(roleRepo)
	postService := services.NewPostService(postRepo)
	commentService := services.NewCommentService(commentRepo, postRepo)
	settingService := services.NewSettingService(settingRepo)

	// 初始化存储后端
	storageSvc, err := storage.NewStorageFromEnv()
	if err != nil {
		log.Fatalf("failed to initialize storage: %v", err)
	}

	imageService := services.NewImageService(imageRepo, storageSvc)

	app := router.SetupRouter(userService, roleService, postService, commentService, settingService, imageService)

	if err := app.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
