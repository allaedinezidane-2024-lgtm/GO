package helper

import (
	"latihan-fiber/app/model"

	"github.com/gofiber/fiber/v2"
)

const LocalsAuthUser = "authUser"

func SetAuthUser(c *fiber.Ctx, user model.AuthUser) {
	c.Locals(LocalsAuthUser, user)
}

func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(LocalsAuthUser).(model.AuthUser)
	return user, ok
}
