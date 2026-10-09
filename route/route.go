package route

import (
	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/service"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
)

func Setup(
	app *fiber.App,
	userService *service.UserService,
	jwtManager *helper.JWTManager,
) {
	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "API is healthy",
		})
	})

	users := api.Group("/users")

	// Public
	users.Post("/", userService.Create)
	users.Post("/login", userService.Login)

	// Protected
	protected := users.Group("", middleware.JWTAuth(jwtManager))

	protected.Get("/", userService.List)
	protected.Get("/:id", userService.Get)
	protected.Put("/:id", userService.Replace)
	protected.Patch("/:id", userService.Patch)
	protected.Delete("/:id", userService.Delete)
}
