package main

import (
	"context"
	"log"
	"os"
	"time"

	"blog-backend/config"
	"blog-backend/internal/healthcheck"
	"blog-backend/models"
	"blog-backend/repositories"
	"blog-backend/router"
	"blog-backend/services"
	"blog-backend/utils/storage"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := healthcheck.Probe(ctx, "http://127.0.0.1:8080/health"); err != nil {
			log.Print(err)
			os.Exit(1)
		}
		return
	}

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

	// 启动时同步 Admin 权限，确保代码中定义的权限已写入数据库
	if err := userService.SyncAdminPermissions(); err != nil {
		log.Printf("warning: failed to sync admin permissions: %v", err)
	}

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

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("failed to obtain database health checker")
	}
	app := router.SetupRouter(userService, roleService, postService, commentService, settingService, imageService, sqlDB.PingContext)

	if err := app.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
