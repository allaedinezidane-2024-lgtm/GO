package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/repository"
	"latihan-fiber/app/service"
	"latihan-fiber/config"
	"latihan-fiber/database"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
	"latihan-fiber/route"
)

func main() {
	config.LoadEnv()

	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("gagal koneksi database: %v", err)
	}
	defer pool.Close()

	roleRepository := repository.NewRoleRepository(pool)
	mapping, err := roleRepository.PermissionsByRole(ctx)
	if err != nil {
		log.Fatalf("gagal memuat permission RBAC: %v; pastikan migrations/003_rbac.sql sudah dijalankan", err)
	}
	permissionSet := helper.NewPermissionSet(mapping)

	log.Printf("RBAC aktif; role terdaftar: %v", permissionSet.KnownRoles())

	userRepository := repository.NewUserRepository(pool)
	jwtManager := helper.NewJWTManager(
		config.GetEnv("JWT_SECRET", ""),
		config.GetEnv("JWT_ISSUER", "latihan-fiber"),
		15*time.Minute,
	)
	userService := service.NewUserService(userRepository, jwtManager, permissionSet)

	studentRepository := repository.NewStudentRepository(pool)
	studentService := service.NewStudentService(studentRepository, permissionSet)

	app := fiber.New()
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.CORS())
	app.Use(middleware.RequireJSON())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true, "message": "API berjalan"})
	})

	route.Setup(app, userService, studentService, jwtManager, permissionSet)

	port := config.GetEnv("APP_PORT", "3000")
	log.Printf("server berjalan di port %s", port)
	log.Fatal(app.Listen(":" + port))
}
