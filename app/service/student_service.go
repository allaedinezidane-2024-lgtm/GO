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

	q := helper.ParseListQuery(c)

	students, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(
			c,
			fiber.StatusInternalServerError,
			"gagal mengambil data mahasiswa",
		)
	}

	hasMore := len(students) > q.Limit
	if hasMore {
		students = students[:q.Limit]
	}

	nextCursor := ""
	if hasMore && len(students) > 0 {
		last := students[len(students)-1]

		nextCursor, err = helper.EncodeCursor(last.CreatedAt, last.ID)
		if err != nil {
			return helper.Fail(
				c,
				fiber.StatusInternalServerError,
				"gagal membuat cursor",
			)
		}
	}

	return helper.SuccessNegotiated(
		c,
		fiber.StatusOK,
		"daftar mahasiswa berhasil diambil",
		fiber.Map{
			"items":       students,
			"limit":       q.Limit,
			"next_cursor": nextCursor,
			"has_more":    hasMore,
		},
	)
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

	validationErrors := helper.ValidateStruct(req)
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

	validationErrors := helper.ValidateStruct(req)
	if len(validationErrors) > 0 {
		return helper.FailValidation(c, validationErrors)
	}

	if req.NIM != nil {
		student.NIM = strings.TrimSpace(*req.NIM)
	}

	if req.Nama != nil {
		student.Nama = strings.TrimSpace(*req.Nama)
	}

	if req.Prodi != nil {
		student.Prodi = strings.TrimSpace(*req.Prodi)
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

func trimPointer(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
