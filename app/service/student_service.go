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

type StudentService struct {
	repo          *repository.StudentRepository
	permissionSet *helper.PermissionSet
}

func NewStudentService(
	repo *repository.StudentRepository,
	permissionSet *helper.PermissionSet,
) *StudentService {
	return &StudentService{
		repo:          repo,
		permissionSet: permissionSet,
	}
}

func (s *StudentService) canAccess(
	current model.AuthUser,
	student *model.Student,
	permission string,
) bool {
	return CanAccessStudent(
		current,
		student.OwnerID,
		s.permissionSet,
		permission,
	)
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	students, err := s.repo.FindAll(ctx)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "daftar mahasiswa berhasil diambil", students)
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	current, exists := helper.CurrentUser(c)
	if !exists {
		return helper.Fail(c, fiber.StatusUnauthorized, "login diperlukan")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	student, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data mahasiswa")
	}

	if !s.canAccess(current, student, "student:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "permission ditolak")
	}

	return helper.Success(c, fiber.StatusOK, "data mahasiswa ditemukan", student)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	current, exists := helper.CurrentUser(c)
	if !exists {
		return helper.Fail(c, fiber.StatusUnauthorized, "login diperlukan")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Nama = strings.TrimSpace(req.Nama)
	req.Prodi = strings.TrimSpace(req.Prodi)

	validationErrors := make(map[string]string)

	if req.NIM == "" {
		validationErrors["nim"] = "NIM wajib diisi"
	} else if len(req.NIM) > 20 {
		validationErrors["nim"] = "NIM maksimal 20 karakter"
	}

	if req.Nama == "" {
		validationErrors["nama"] = "nama wajib diisi"
	} else if len(req.Nama) > 100 {
		validationErrors["nama"] = "nama maksimal 100 karakter"
	}

	if req.Prodi == "" {
		validationErrors["prodi"] = "program studi wajib diisi"
	} else if len(req.Prodi) > 100 {
		validationErrors["prodi"] = "program studi maksimal 100 karakter"
	}

	if len(validationErrors) > 0 {
		return helper.FailValidation(c, validationErrors)
	}

	student := model.Student{
		NIM:     req.NIM,
		Nama:    req.Nama,
		Prodi:   req.Prodi,
		OwnerID: current.ID,
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	err := s.repo.Create(ctx, &student)
	if isStudentDuplicate(err) {
		return helper.Fail(c, fiber.StatusConflict, "NIM sudah terdaftar")
	}
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal membuat data mahasiswa")
	}

	return helper.Created(
		c,
		"data mahasiswa berhasil dibuat",
		student,
		"/api/v1/students/"+strconv.Itoa(student.ID),
	)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	current, exists := helper.CurrentUser(c)
	if !exists {
		return helper.Fail(c, fiber.StatusUnauthorized, "login diperlukan")
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if req.NIM == nil && req.Nama == nil && req.Prodi == nil {
		return helper.Fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	student, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data mahasiswa")
	}

	if !s.canAccess(current, student, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "permission ditolak")
	}

	if req.NIM != nil {
		value := strings.TrimSpace(*req.NIM)
		if value == "" || len(value) > 20 {
			return helper.FailValidation(c, map[string]string{
				"nim": "NIM wajib diisi dan maksimal 20 karakter",
			})
		}
		student.NIM = value
	}

	if req.Nama != nil {
		value := strings.TrimSpace(*req.Nama)
		if value == "" || len(value) > 100 {
			return helper.FailValidation(c, map[string]string{
				"nama": "nama wajib diisi dan maksimal 100 karakter",
			})
		}
		student.Nama = value
	}

	if req.Prodi != nil {
		value := strings.TrimSpace(*req.Prodi)
		if value == "" || len(value) > 100 {
			return helper.FailValidation(c, map[string]string{
				"prodi": "program studi wajib diisi dan maksimal 100 karakter",
			})
		}
		student.Prodi = value
	}

	err = s.repo.Update(ctx, student)
	if isStudentDuplicate(err) {
		return helper.Fail(c, fiber.StatusConflict, "NIM sudah terdaftar")
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui data mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "data mahasiswa berhasil diperbarui", student)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	current, exists := helper.CurrentUser(c)
	if !exists {
		return helper.Fail(c, fiber.StatusUnauthorized, "login diperlukan")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	student, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data mahasiswa")
	}

	if !s.canAccess(current, student, "student:delete") {
		return helper.Fail(c, fiber.StatusForbidden, "permission ditolak")
	}

	err = s.repo.Delete(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return helper.Fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal menghapus data mahasiswa")
	}

	return helper.NoContent(c)
}

func isStudentDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
