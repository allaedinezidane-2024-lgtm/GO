package service

import (
	"regexp"
	"strings"

	"latihan-fiber/app/model"
)

func ValidateCreate(user model.CreateUserRequest) map[string]string {
	errors := make(map[string]string)

	if strings.TrimSpace(user.Username) == "" {
		errors["username"] = "username is required"
	}

	if strings.TrimSpace(user.Email) == "" {
		errors["email"] = "email is required"
	} else if !isValidEmail(user.Email) {
		errors["email"] = "email must be valid"
	}

	if strings.TrimSpace(user.Password) == "" {
		errors["password"] = "password is required"
	}

	return errors
}

func ValidateReplace(user model.ReplaceUserRequest) map[string]string {
	errors := make(map[string]string)

	if strings.TrimSpace(user.Username) == "" {
		errors["username"] = "username is required"
	}

	if strings.TrimSpace(user.Email) == "" {
		errors["email"] = "email is required"
	} else if !isValidEmail(user.Email) {
		errors["email"] = "email must be valid"
	}

	return errors
}

func ApplyPatch(user *model.User, patch model.PatchUserRequest) map[string]string {
	errors := make(map[string]string)

	if patch.Username != nil {
		if strings.TrimSpace(*patch.Username) == "" {
			errors["username"] = "username is required"
		} else {
			user.Username = *patch.Username
		}
	}

	if patch.Email != nil {
		if !isValidEmail(*patch.Email) {
			errors["email"] = "email must be valid"
		} else {
			user.Email = *patch.Email
		}
	}

	if patch.IsActive != nil {
		user.IsActive = *patch.IsActive
	}

	return errors
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

func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`

	return regexp.MustCompile(pattern).MatchString(email)
}
