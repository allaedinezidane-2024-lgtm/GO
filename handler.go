package main

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
)

var userRepo *repository.UserRepository

func listUsers(c *fiber.Ctx) error {
	q := parseListQuery(c)

	users, total, err := userRepo.FindAll(context.Background(), q)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "gagal mengambil data user")
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}

	return c.JSON(model.WebResponse{
		Success: true,
		Message: "daftar user berhasil diambil",
		Data:    users,
		Meta: &model.Meta{
			Page:       q.Page,
			Limit:      q.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func getUser(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	user, err := userRepo.FindByID(context.Background(), id)

	if errors.Is(err, pgx.ErrNoRows) {
		return fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "gagal mengambil user")
	}

	return ok(c, "user ditemukan", user)
}

func createUser(c *fiber.Ctx) error {
	var req model.CreateUserRequest

	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	errs := map[string]string{}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if req.Username == "" {
		errs["username"] = "wajib diisi"
	}

	if !strings.Contains(req.Email, "@") {
		errs["email"] = "format email tidak valid"
	}

	if len(req.Password) < 8 {
		errs["password"] = "minimal 8 karakter"
	}

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		IsActive: true,
	}

	err := userRepo.Create(context.Background(), user)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fail(c, fiber.StatusConflict, "username sudah dipakai")
		}

		return fail(c, fiber.StatusInternalServerError, "gagal membuat user")
	}

	return created(
		c,
		"user berhasil dibuat",
		user,
		"/api/v1/users/"+strconv.Itoa(user.ID),
	)
}

func replaceUser(c *fiber.Ctx) error {
	id, valid := paramID(c)

	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.ReplaceUserRequest

	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	errs := map[string]string{}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if req.Username == "" {
		errs["username"] = "wajib diisi pada PUT"
	}

	if !strings.Contains(req.Email, "@") {
		errs["email"] = "wajib diisi dan berformat email pada PUT"
	}

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	user, err := userRepo.FindByID(context.Background(), id)

	if errors.Is(err, pgx.ErrNoRows) {
		return fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "gagal mengambil user")
	}

	user.Username = req.Username
	user.Email = req.Email
	user.IsActive = req.IsActive

	err = userRepo.Update(context.Background(), user)

	if errors.Is(err, pgx.ErrNoRows) {
		return fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fail(c, fiber.StatusConflict, "username sudah dipakai")
		}

		return fail(c, fiber.StatusInternalServerError, "gagal memperbarui user")
	}

	return ok(c, "user berhasil diganti seluruhnya", user)
}

func patchUser(c *fiber.Ctx) error {
	id, valid := paramID(c)

	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.PatchUserRequest

	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if req.Username == nil && req.Email == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	user, err := userRepo.FindByID(context.Background(), id)

	if errors.Is(err, pgx.ErrNoRows) {
		return fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "gagal mengambil user")
	}

	if req.Username != nil {
		username := strings.TrimSpace(*req.Username)

		if username == "" {
			return failValidation(c, map[string]string{
				"username": "tidak boleh kosong",
			})
		}

		user.Username = username
	}

	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)

		if !strings.Contains(email, "@") {
			return failValidation(c, map[string]string{
				"email": "format email tidak valid",
			})
		}

		user.Email = email
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	err = userRepo.Update(context.Background(), user)

	if errors.Is(err, pgx.ErrNoRows) {
		return fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fail(c, fiber.StatusConflict, "username sudah dipakai")
		}

		return fail(c, fiber.StatusInternalServerError, "gagal memperbarui user")
	}

	return ok(c, "user berhasil diperbarui sebagian", user)
}

func deleteUser(c *fiber.Ctx) error {
	id, valid := paramID(c)

	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	err := userRepo.Delete(context.Background(), id)

	if errors.Is(err, pgx.ErrNoRows) {
		return fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "gagal menghapus user")
	}

	return noContent(c)
}
