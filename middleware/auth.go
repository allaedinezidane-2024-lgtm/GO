package middleware

import (
	"latihan-fiber/helper"

	"github.com/gofiber/fiber/v2"
)

func JWTAuth(jwtManager *helper.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")

		if authHeader == "" {
			return helper.Fail(
				c,
				fiber.StatusUnauthorized,
				"Authorization header required",
			)
		}

		token, err := helper.ExtractBearerToken(authHeader)
		if err != nil {
			return helper.Fail(
				c,
				fiber.StatusUnauthorized,
				"Invalid authorization header",
			)
		}

		user, err := jwtManager.Parse(token)
		if err != nil {
			return helper.Fail(
				c,
				fiber.StatusUnauthorized,
				"Invalid or expired token",
			)
		}

		helper.SetAuthUser(c, user)

		return c.Next()
	}
}
