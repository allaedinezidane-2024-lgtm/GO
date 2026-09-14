package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"latihan-fiber/app/repository"
	"latihan-fiber/config"
	"latihan-fiber/database"
)

func requireJSON(c *fiber.Ctx) error {
	if c.Method() == fiber.MethodPost ||
		c.Method() == fiber.MethodPut ||
		c.Method() == fiber.MethodPatch {

		contentType := c.Get("Content-Type")

		if contentType == "" ||
			contentType[:min(len(contentType), 16)] != "application/json" {
			return fail(
				c,
				fiber.StatusUnsupportedMediaType,
				"Content-Type harus application/json",
			)
		}
	}

	return c.Next()
}

func main() {
	// Load .env
	config.LoadEnv()

	// Koneksi PostgreSQL
	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// Repository
	userRepo = repository.NewUserRepository(pool)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return fail(c, fiber.StatusInternalServerError, err.Error())
		},
	})

	// Middleware
	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(cors.New())
	app.Use(requireJSON)

	// Endpoint Modul 1
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// API v1
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "API dan database berjalan",
		})
	})

	// Users
	usersRoute := api.Group("/users")

	usersRoute.Get("/", listUsers)
	usersRoute.Get("/:id", getUser)
	usersRoute.Post("/", createUser)
	usersRoute.Put("/:id", replaceUser)
	usersRoute.Patch("/:id", patchUser)
	usersRoute.Delete("/:id", deleteUser)

	log.Fatal(app.Listen(":" + config.GetEnv("APP_PORT", "3000")))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
