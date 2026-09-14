package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func requireJSON(c *fiber.Ctx) error {
	if c.Method() == fiber.MethodPost ||
		c.Method() == fiber.MethodPut ||
		c.Method() == fiber.MethodPatch {

		contentType := c.Get("Content-Type")

		if contentType == "" ||
			contentType[:min(len(contentType), 16)] != "application/json" {
			return fail(c, fiber.StatusUnsupportedMediaType,
				"Content-Type harus application/json")
		}
	}

	return c.Next()
}

func main() {
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

	// Tetap pertahankan endpoint Modul 1
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// API v1
	api := app.Group("/api/v1")

	usersRoute := api.Group("/users")

	usersRoute.Get("/", listUsers)
	usersRoute.Get("/:id", getUser)
	usersRoute.Post("/", createUser)
	usersRoute.Put("/:id", replaceUser)
	usersRoute.Patch("/:id", patchUser)
	usersRoute.Delete("/:id", deleteUser)

	log.Fatal(app.Listen(":3000"))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
