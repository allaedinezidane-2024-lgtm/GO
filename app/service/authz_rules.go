package service

import (
	"latihan-fiber/app/model"
	"latihan-fiber/helper"
)

func CanAccessUser(current model.AuthUser, targetID int, permissionSet *helper.PermissionSet, anyPermission string) bool {
	if current.ID == targetID {
		return true
	}

	return permissionSet != nil && permissionSet.Can(current.Role, anyPermission)
}
