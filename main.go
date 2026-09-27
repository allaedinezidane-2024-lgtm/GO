package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/repository"
	"latihan-fiber/app/service"
	"latihan-fiber/config"
	"latihan-fiber/database"
	"latihan-fiber/middleware"
	"latihan-fiber/route"
)

func main() {
	// Load environment
	config.LoadEnv()

	// Database
	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("gagal koneksi database: %v", err)
	}
	defer pool.Close()

	// Repository
	userRepository := repository.NewUserRepository(pool)

	// Service
	userService := service.NewUserService(userRepository)

	// Fiber app
	app := fiber.New()

	// Middleware
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.CORS())
	app.Use(middleware.RequireJSON())

	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "API berjalan",
		})
	})

	// Routes
	route.Setup(app, userService)

	// Run server
	port := config.GetEnv("APP_PORT", "3000")

	log.Printf("server berjalan di port %s", port)

	log.Fatal(app.Listen(":" + port))
}
