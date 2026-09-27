package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"latihan-fiber/helper"
)

func RequestID() fiber.Handler {
	return requestid.New()
}

func Logger() fiber.Handler {
	return logger.New()
}

func CORS() fiber.Handler {
	return cors.New()
}

func RequireJSON() fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch:
			contentType := c.Get("Content-Type")

			if contentType == "" ||
				!strings.HasPrefix(
					strings.ToLower(contentType),
					"application/json",
				) {
				return helper.Fail(
					c,
					fiber.StatusUnsupportedMediaType,
					"Content-Type harus application/json",
				)
			}
		}

		return c.Next()
	}
}
