package route

import (
	"latihan-fiber/app/service"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupSiakad(app *fiber.App, s *service.SiakadService, jwt *helper.JWTManager) {
	api := app.Group("/api/v1")
	api.Post("/auth/login", s.Login)

	protected := api.Group("", middleware.JWTAuth(jwt))
	protected.Get("/auth/me", s.Me)
	protected.Get("/students", s.ListStudents)
	protected.Post("/students", s.CreateStudent)
	protected.Get("/students/:id", s.GetStudent)
	protected.Put("/students/:id", s.UpdateStudent)
	protected.Delete("/students/:id", s.DeleteStudent)
	protected.Get("/courses", s.ListCourses)
	protected.Post("/enrollments", s.CreateEnrollment)
	protected.Delete("/enrollments/:id", s.DeleteEnrollment)
}
