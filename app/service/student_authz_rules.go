package service

import (
	"latihan-fiber/app/model"
	"latihan-fiber/helper"
)

func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if current.ID == ownerID {
		return true
	}

	return perms != nil && perms.Can(current.Role, anyPermission)
}
