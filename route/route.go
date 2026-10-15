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
	studentService *service.StudentService,
	jwtManager *helper.JWTManager,
	permissionSet *helper.PermissionSet,
) {
	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "API is healthy",
		})
	})

	users := api.Group("/users")
	users.Post("/", userService.Create)
	users.Post("/login", userService.Login)

	protected := users.Group("", middleware.JWTAuth(jwtManager))
	protected.Get(
		"/",
		middleware.RequirePermission(permissionSet, "user:list"),
		userService.List,
	)
	protected.Get("/:id", userService.Get)
	protected.Put("/:id", userService.Replace)
	protected.Patch("/:id", userService.Patch)
	protected.Delete(
		"/:id",
		middleware.RequirePermission(permissionSet, "user:delete"),
		userService.Delete,
	)
	protected.Patch(
		"/:id/role",
		middleware.RequirePermission(permissionSet, "role:assign"),
		userService.AssignRole,
	)

	students := api.Group(
		"/students",
		middleware.JWTAuth(jwtManager),
	)

	students.Post(
		"/",
		middleware.RequirePermission(permissionSet, "student:create"),
		studentService.Create,
	)
	students.Get(
		"/",
		middleware.RequirePermission(permissionSet, "student:list"),
		studentService.List,
	)
	students.Get("/:id", studentService.Get)
	students.Patch("/:id", studentService.Patch)
	students.Delete("/:id", studentService.Delete)
}
