package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type UserService struct {
	repo repository.UserRepositoryInterface
}

func NewUserService(repo repository.UserRepositoryInterface) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseListQuery(c)

	users, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data user")
	}

	meta := &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: CountTotalPages(total, q.Limit),
	}

	return helper.SuccessList(c, "daftar user berhasil diambil", users, meta)
}

func (s *UserService) Get(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, err := s.repo.FindByID(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil user")
	}

	return helper.Success(c, fiber.StatusOK, "user ditemukan", user)
}

func (s *UserService) Create(c *fiber.Ctx) error {
	var req model.CreateUserRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if errs := ValidateCreate(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	user := model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		IsActive: true,
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	err := s.repo.Create(ctx, &user)

	if isDuplicateError(err) {
		return helper.Fail(c, fiber.StatusConflict, "username sudah dipakai")
	}

	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal membuat user")
	}

	location := "/api/v1/users/" + strconv.Itoa(user.ID)

	return helper.Created(c, "user berhasil dibuat", user, location)
}

func (s *UserService) Replace(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.ReplaceUserRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if errs := ValidateReplace(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, err := s.repo.FindByID(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil user")
	}

	user.Username = req.Username
	user.Email = req.Email
	user.IsActive = req.IsActive

	err = s.repo.Update(ctx, user)

	if isDuplicateError(err) {
		return helper.Fail(c, fiber.StatusConflict, "username sudah dipakai")
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui user")
	}

	return helper.Success(c, fiber.StatusOK, "user berhasil diganti seluruhnya", user)
}

func (s *UserService) Patch(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.PatchUserRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if IsEmptyPatch(req) {
		return helper.Fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, err := s.repo.FindByID(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil user")
	}

	errs := ApplyPatch(user, req)

	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	updatedUser := *user

	err = s.repo.Update(ctx, &updatedUser)

	if isDuplicateError(err) {
		return helper.Fail(c, fiber.StatusConflict, "username sudah dipakai")
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui user")
	}

	return helper.Success(c, fiber.StatusOK, "user berhasil diperbarui sebagian", updatedUser)
}

func (s *UserService) Delete(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	err := s.repo.Delete(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal menghapus user")
	}

	return helper.NoContent(c)
}

func isDuplicateError(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
