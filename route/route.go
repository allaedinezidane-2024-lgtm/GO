package route

import (
	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/service"
)

func Setup(app *fiber.App, userService *service.UserService) {
	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "API is healthy",
		})
	})

	users := api.Group("/users")

	users.Get("/", userService.List)
	users.Get("/:id", userService.Get)
	users.Post("/", userService.Create)
	users.Put("/:id", userService.Replace)
	users.Patch("/:id", userService.Patch)
	users.Delete("/:id", userService.Delete)
}
