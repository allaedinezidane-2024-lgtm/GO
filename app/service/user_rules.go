package service

import (
	"strings"

	"latihan-fiber/app/model"
	"latihan-fiber/helper"
)

func ValidateCreate(user model.CreateUserRequest) map[string]string {
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(user.Email)

	return helper.ValidateStruct(user)
}

func ValidateReplace(user model.ReplaceUserRequest) map[string]string {
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(user.Email)

	return helper.ValidateStruct(user)
}

func ApplyPatch(user *model.User, patch model.PatchUserRequest) map[string]string {
	validationErrors := helper.ValidateStruct(patch)
	if len(validationErrors) > 0 {
		return validationErrors
	}

	if patch.Username != nil {
		value := strings.TrimSpace(*patch.Username)
		if value == "" {
			return map[string]string{
				"username": "field wajib diisi",
			}
		}
		user.Username = value
	}

	if patch.Email != nil {
		value := strings.TrimSpace(*patch.Email)
		if value == "" {
			return map[string]string{
				"email": "field wajib diisi",
			}
		}
		user.Email = value
	}

	if patch.IsActive != nil {
		user.IsActive = *patch.IsActive
	}

	return nil
}

func IsEmptyPatch(patch model.PatchUserRequest) bool {
	return patch.Username == nil &&
		patch.Email == nil &&
		patch.IsActive == nil
}

func CountTotalPages(total, limit int) int {
	if limit <= 0 || total == 0 {
		return 0
	}

	return (total + limit - 1) / limit
}
