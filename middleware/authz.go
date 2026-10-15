package middleware

import (
	"latihan-fiber/helper"

	"github.com/gofiber/fiber/v2"
)

func RequirePermission(permissionSet *helper.PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Fail(c, fiber.StatusUnauthorized, "login diperlukan")
		}

		if permissionSet == nil || !permissionSet.Can(user.Role, permission) {
			return helper.Fail(c, fiber.StatusForbidden, "permission ditolak")
		}

		return c.Next()
	}
}

func RequireRole(permissionSet *helper.PermissionSet, roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Fail(c, fiber.StatusUnauthorized, "login diperlukan")
		}

		if permissionSet == nil || !permissionSet.IsKnownRole(user.Role) {
			return helper.Fail(c, fiber.StatusForbidden, "role tidak dikenal")
		}

		for _, role := range roles {
			if user.Role == role {
				return c.Next()
			}
		}

		return helper.Fail(c, fiber.StatusForbidden, "role ditolak")
	}
}
