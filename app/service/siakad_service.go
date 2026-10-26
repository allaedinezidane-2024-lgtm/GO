package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"latihan-fiber/app/model"
	"latihan-fiber/helper"
)

type SiakadService struct {
	db       *pgxpool.Pool
	jwt      *helper.JWTManager
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func NewSiakadService(db *pgxpool.Pool, jwt *helper.JWTManager) *SiakadService {
	return &SiakadService{db: db, jwt: jwt, attempts: make(map[string][]time.Time)}
}

var nimPattern = regexp.MustCompile(`^\d{12}$`)
var yearPattern = regexp.MustCompile(`^\d{4}/\d{4}-(Ganjil|Genap)$`)

func (s *SiakadService) isAdmin(c *fiber.Ctx) (model.AuthUser, error) {
	u, ok := helper.CurrentUser(c)
	if !ok {
		return u, helper.Fail(c, 401, "Token pengguna tidak valid")
	}
	if u.Role != "admin" {
		return u, helper.Fail(c, 403, "Endpoint ini hanya untuk admin")
	}
	return u, nil
}

func (s *SiakadService) studentForUser(ctx context.Context, userID int) (int, float64, error) {
	var id int
	var ipk float64
	err := s.db.QueryRow(ctx,
		`SELECT id, ipk_terakhir FROM students
         WHERE user_id=$1 AND deleted_at IS NULL`, userID,
	).Scan(&id, &ipk)
	return id, ipk, err
}

func (s *SiakadService) Login(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, 400, "Format JSON tidak valid")
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || len(req.Password) < 8 {
		return helper.FailValidation(c, map[string]string{
			"email": "Email wajib diisi", "password": "Password minimal 8 karakter",
		})
	}

	ip := c.IP()
	now := time.Now()
	s.mu.Lock()
	recent := make([]time.Time, 0, 5)
	for _, t := range s.attempts[ip] {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	s.attempts[ip] = recent
	if len(recent) >= 5 {
		s.mu.Unlock()
		return helper.Fail(c, 429, "Terlalu banyak percobaan login. Coba lagi satu menit.")
	}
	s.mu.Unlock()

	var id int
	var email, passwordHash, role string
	var active bool
	err := s.db.QueryRow(c.Context(),
		`SELECT id,email,password,role,is_active FROM users WHERE LOWER(email)=LOWER($1)`,
		req.Email,
	).Scan(&id, &email, &passwordHash, &role, &active)

	valid := err == nil && active && bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) == nil
	if valid && role == "mahasiswa" {
		var deletedAt *time.Time
		err = s.db.QueryRow(c.Context(),
			`SELECT deleted_at FROM students WHERE user_id=$1`, id,
		).Scan(&deletedAt)
		valid = err == nil && deletedAt == nil
	}
	if !valid {
		s.mu.Lock()
		s.attempts[ip] = append(s.attempts[ip], now)
		s.mu.Unlock()
		return helper.Fail(c, 401, "Email atau password salah")
	}

	s.mu.Lock()
	delete(s.attempts, ip)
	s.mu.Unlock()

	token, err := s.jwt.GenerateAccess(model.AuthUser{
		ID: id, Username: email, Email: email, Role: role,
	})
	if err != nil {
		return helper.Fail(c, 500, "Gagal membuat token")
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Login berhasil",
		"data": fiber.Map{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_in":   int(s.jwt.AccessTTL().Seconds()),
			"user":         fiber.Map{"id": id, "email": email, "role": role},
		},
	})
}

func (s *SiakadService) Me(c *fiber.Ctx) error {
	u, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, 401, "Token tidak valid")
	}
	var id int
	var email, role string
	var active bool
	err := s.db.QueryRow(c.Context(),
		`SELECT id,email,role,is_active FROM users WHERE id=$1`, u.ID,
	).Scan(&id, &email, &role, &active)
	if err != nil || !active {
		return helper.Fail(c, 401, "Pengguna tidak aktif atau tidak ditemukan")
	}
	data := fiber.Map{"id": id, "email": email, "role": role}
	if role == "mahasiswa" {
		var nim, nama, prodi string
		var angkatan int
		var ipk float64
		err = s.db.QueryRow(c.Context(),
			`SELECT nim,nama,prodi,angkatan,ipk_terakhir
             FROM students WHERE user_id=$1 AND deleted_at IS NULL`, id,
		).Scan(&nim, &nama, &prodi, &angkatan, &ipk)
		if err != nil {
			return helper.Fail(c, 404, "Data mahasiswa tidak ditemukan")
		}
		data["student"] = fiber.Map{
			"nim": nim, "nama": nama, "prodi": prodi,
			"angkatan": angkatan, "ipk_terakhir": ipk,
		}
	}
	return helper.Success(c, 200, "Profil pengguna", data)
}

func (s *SiakadService) ListStudents(c *fiber.Ctx) error {
	if _, err := s.isAdmin(c); err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "10"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 50 {
		perPage = 50
	}

	prodi := strings.TrimSpace(c.Query("prodi"))
	search := strings.TrimSpace(c.Query("search"))
	angkatan, _ := strconv.Atoi(c.Query("angkatan", "0"))

	sortBy := "st.nama ASC"
	switch c.Query("sort", "nama") {
	case "nama":
		sortBy = "st.nama ASC"
	case "-ipk_terakhir":
		sortBy = "st.ipk_terakhir DESC, st.nama ASC"
	default:
		return helper.FailValidation(c, map[string]string{
			"sort": "Gunakan nama atau -ipk_terakhir",
		})
	}

	var total int
	err := s.db.QueryRow(c.Context(), `
        SELECT COUNT(*)
        FROM students
        WHERE deleted_at IS NULL
          AND ($1 = '' OR prodi ILIKE '%' || $1 || '%')
          AND ($2 = 0 OR angkatan = $2)
          AND ($3 = '' OR nim ILIKE '%' || $3 || '%' OR nama ILIKE '%' || $3 || '%')
    `, prodi, angkatan, search).Scan(&total)
	if err != nil {
		fmt.Printf("ListStudents COUNT error: %v\n", err)
		return helper.Fail(c, 500, "Gagal mengambil data mahasiswa")
	}

	query := fmt.Sprintf(`
        SELECT st.id, st.nim, st.nama, u.email, st.prodi,
               st.angkatan, st.ipk_terakhir, st.created_at
        FROM students st
        LEFT JOIN users u ON u.id = st.user_id
        WHERE st.deleted_at IS NULL
          AND ($1 = '' OR st.prodi ILIKE '%%' || $1 || '%%')
          AND ($2 = 0 OR st.angkatan = $2)
          AND ($3 = '' OR st.nim ILIKE '%%' || $3 || '%%'
               OR st.nama ILIKE '%%' || $3 || '%%')
        ORDER BY %s
        LIMIT $4 OFFSET $5
    `, sortBy)

	rows, err := s.db.Query(
		c.Context(), query, prodi, angkatan, search,
		perPage, (page-1)*perPage,
	)
	if err != nil {
		fmt.Printf("ListStudents query error: %v\n", err)
		return helper.Fail(c, 500, "Gagal mengambil data mahasiswa")
	}
	defer rows.Close()

	items := make([]fiber.Map, 0)
	for rows.Next() {
		var id, angkatan int
		var nim, nama, prodi string
		var email *string
		var ipk float64
		var created time.Time

		if err := rows.Scan(
			&id, &nim, &nama, &email, &prodi,
			&angkatan, &ipk, &created,
		); err != nil {
			fmt.Printf("ListStudents scan error: %v\n", err)
			return helper.Fail(c, 500, "Gagal membaca data mahasiswa")
		}

		emailValue := ""
		if email != nil {
			emailValue = *email
		}

		items = append(items, fiber.Map{
			"id":           id,
			"nim":          nim,
			"nama":         nama,
			"email":        emailValue,
			"prodi":        prodi,
			"angkatan":     angkatan,
			"ipk_terakhir": ipk,
			"created_at":   created,
		})
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("ListStudents rows error: %v\n", err)
		return helper.Fail(c, 500, "Gagal membaca data mahasiswa")
	}

	lastPage := (total + perPage - 1) / perPage
	if lastPage == 0 {
		lastPage = 1
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Daftar mahasiswa",
		"data":    items,
		"meta": fiber.Map{
			"current_page": page,
			"per_page":     perPage,
			"total":        total,
			"last_page":    lastPage,
		},
	})
}
func (s *SiakadService) CreateStudent(c *fiber.Ctx) error {
	admin, err := s.isAdmin(c)
	if err != nil {
		return err
	}
	var req struct {
		NIM      string  `json:"nim"`
		Nama     string  `json:"nama"`
		Email    string  `json:"email"`
		Prodi    string  `json:"prodi"`
		Angkatan int     `json:"angkatan"`
		IPK      float64 `json:"ipk_terakhir"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, 400, "Format JSON tidak valid")
	}
	req.NIM = strings.TrimSpace(req.NIM)
	req.Nama = strings.TrimSpace(req.Nama)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Prodi = strings.TrimSpace(req.Prodi)
	errs := map[string]string{}
	if !nimPattern.MatchString(req.NIM) {
		errs["nim"] = "NIM harus tepat 12 digit angka"
	}
	if req.Nama == "" || len(req.Nama) > 100 {
		errs["nama"] = "Nama wajib diisi, maksimal 100 karakter"
	}
	if !strings.Contains(req.Email, "@") || len(req.Email) > 255 {
		errs["email"] = "Email tidak valid"
	}
	if req.Prodi == "" || len(req.Prodi) > 100 {
		errs["prodi"] = "Program studi wajib diisi, maksimal 100 karakter"
	}
	if req.Angkatan < 1000 || req.Angkatan > time.Now().Year() {
		errs["angkatan"] = "Angkatan harus empat digit dan tidak melebihi tahun ini"
	}
	if req.IPK < 0 || req.IPK > 4 {
		errs["ipk_terakhir"] = "IPK harus antara 0 sampai 4"
	}
	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NIM), bcrypt.DefaultCost)
	if err != nil {
		return helper.Fail(c, 500, "Gagal membuat password mahasiswa")
	}

	tx, err := s.db.Begin(c.Context())
	if err != nil {
		return helper.Fail(c, 500, "Gagal memulai transaksi")
	}
	defer tx.Rollback(c.Context())

	var userID, studentID int
	err = tx.QueryRow(c.Context(),
		`INSERT INTO users(username,email,password,is_active,role)
         VALUES($1,$2,$3,TRUE,'mahasiswa') RETURNING id`,
		req.NIM, req.Email, string(hash),
	).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			return helper.Fail(c, 422, "NIM atau email sudah digunakan")
		}
		return helper.Fail(c, 500, "Gagal membuat akun mahasiswa")
	}
	err = tx.QueryRow(c.Context(),
		`INSERT INTO students(nim,nama,prodi,owner_id,user_id,angkatan,ipk_terakhir)
         VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		req.NIM, req.Nama, req.Prodi, admin.ID, userID, req.Angkatan, req.IPK,
	).Scan(&studentID)
	if err != nil {
		if isUniqueViolation(err) {
			return helper.Fail(c, 422, "NIM sudah digunakan")
		}
		return helper.Fail(c, 500, "Gagal membuat data mahasiswa")
	}
	if err := tx.Commit(c.Context()); err != nil {
		return helper.Fail(c, 500, "Gagal menyimpan mahasiswa")
	}
	return helper.Created(c, "Mahasiswa berhasil dibuat", fiber.Map{
		"id": studentID, "user_id": userID, "nim": req.NIM, "nama": req.Nama,
		"email": req.Email, "prodi": req.Prodi, "angkatan": req.Angkatan, "ipk_terakhir": req.IPK,
	}, fmt.Sprintf("/api/v1/students/%d", studentID))
}

func (s *SiakadService) GetStudent(c *fiber.Ctx) error {
	u, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, 401, "Token tidak valid")
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return helper.Fail(c, 400, "ID mahasiswa tidak valid")
	}

	var userID, angkatan int
	var nim, nama, prodi string
	var ipk float64
	var created time.Time
	err = s.db.QueryRow(c.Context(),
		`SELECT user_id,nim,nama,prodi,angkatan,ipk_terakhir,created_at
         FROM students WHERE id=$1 AND deleted_at IS NULL`, id,
	).Scan(&userID, &nim, &nama, &prodi, &angkatan, &ipk, &created)
	if err != nil {
		return helper.Fail(c, 404, "Mahasiswa tidak ditemukan")
	}
	if u.Role != "admin" && !(u.Role == "mahasiswa" && userID == u.ID) {
		return helper.Fail(c, 403, "Kamu tidak boleh mengakses data mahasiswa ini")
	}

	batas := 18
	if ipk >= 3 {
		batas = 24
	} else if ipk >= 2.5 {
		batas = 21
	}

	rows, err := s.db.Query(c.Context(),
		`SELECT e.id,c.id,c.kode_mk,c.nama_mk,c.sks,c.semester,e.tahun_akademik
         FROM enrollments e JOIN courses c ON c.id=e.course_id
         WHERE e.student_id=$1 ORDER BY e.tahun_akademik DESC,c.kode_mk`, id)
	if err != nil {
		return helper.Fail(c, 500, "Gagal mengambil KRS mahasiswa")
	}
	defer rows.Close()
	courses := make([]fiber.Map, 0)
	totalSKS := 0
	for rows.Next() {
		var enrollmentID, courseID, sks, semester int
		var kode, namaMK, tahun string
		if err := rows.Scan(&enrollmentID, &courseID, &kode, &namaMK, &sks, &semester, &tahun); err != nil {
			return helper.Fail(c, 500, "Gagal membaca KRS mahasiswa")
		}
		totalSKS += sks
		courses = append(courses, fiber.Map{
			"enrollment_id": enrollmentID, "course_id": courseID, "kode_mk": kode,
			"nama_mk": namaMK, "sks": sks, "semester": semester, "tahun_akademik": tahun,
		})
	}
	return helper.Success(c, 200, "Detail mahasiswa", fiber.Map{
		"id": id, "user_id": userID, "nim": nim, "nama": nama, "prodi": prodi,
		"angkatan": angkatan, "ipk_terakhir": ipk, "created_at": created,
		"courses": courses, "total_sks": totalSKS, "batas_sks": batas,
	})
}

func (s *SiakadService) UpdateStudent(c *fiber.Ctx) error {
	if _, err := s.isAdmin(c); err != nil {
		return err
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return helper.Fail(c, 400, "ID mahasiswa tidak valid")
	}
	var req struct {
		Nama     string  `json:"nama"`
		Prodi    string  `json:"prodi"`
		Angkatan int     `json:"angkatan"`
		IPK      float64 `json:"ipk_terakhir"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, 400, "Format JSON tidak valid")
	}
	req.Nama = strings.TrimSpace(req.Nama)
	req.Prodi = strings.TrimSpace(req.Prodi)
	errs := map[string]string{}
	if req.Nama == "" || len(req.Nama) > 100 {
		errs["nama"] = "Nama wajib diisi, maksimal 100 karakter"
	}
	if req.Prodi == "" || len(req.Prodi) > 100 {
		errs["prodi"] = "Program studi wajib diisi, maksimal 100 karakter"
	}
	if req.Angkatan < 1000 || req.Angkatan > time.Now().Year() {
		errs["angkatan"] = "Angkatan tidak valid"
	}
	if req.IPK < 0 || req.IPK > 4 {
		errs["ipk_terakhir"] = "IPK harus antara 0 sampai 4"
	}
	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}
	tag, err := s.db.Exec(c.Context(),
		`UPDATE students SET nama=$1,prodi=$2,angkatan=$3,ipk_terakhir=$4
         WHERE id=$5 AND deleted_at IS NULL`, req.Nama, req.Prodi, req.Angkatan, req.IPK, id)
	if err != nil {
		return helper.Fail(c, 500, "Gagal memperbarui mahasiswa")
	}
	if tag.RowsAffected() == 0 {
		return helper.Fail(c, 404, "Mahasiswa tidak ditemukan")
	}
	return helper.Success(c, 200, "Mahasiswa berhasil diperbarui", fiber.Map{
		"id": id, "nama": req.Nama, "prodi": req.Prodi, "angkatan": req.Angkatan, "ipk_terakhir": req.IPK,
	})
}

func (s *SiakadService) DeleteStudent(c *fiber.Ctx) error {
	if _, err := s.isAdmin(c); err != nil {
		return err
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return helper.Fail(c, 400, "ID mahasiswa tidak valid")
	}
	tx, err := s.db.Begin(c.Context())
	if err != nil {
		return helper.Fail(c, 500, "Gagal memulai transaksi")
	}
	defer tx.Rollback(c.Context())
	var userID *int
	err = tx.QueryRow(c.Context(),
		`UPDATE students SET deleted_at=NOW()
         WHERE id=$1 AND deleted_at IS NULL RETURNING user_id`, id,
	).Scan(&userID)
	if err == pgx.ErrNoRows {
		return helper.Fail(c, 404, "Mahasiswa tidak ditemukan")
	}
	if err != nil {
		return helper.Fail(c, 500, "Gagal menghapus mahasiswa")
	}
	if userID != nil {
		_, err = tx.Exec(c.Context(), `UPDATE users SET is_active=FALSE WHERE id=$1`, *userID)
		if err != nil {
			return helper.Fail(c, 500, "Gagal menonaktifkan akun mahasiswa")
		}
	}
	if err = tx.Commit(c.Context()); err != nil {
		return helper.Fail(c, 500, "Gagal menyimpan perubahan")
	}
	return helper.Success(c, 200, "Mahasiswa berhasil dihapus", fiber.Map{"id": id})
}

func (s *SiakadService) ListCourses(c *fiber.Ctx) error {
	u, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, 401, "Token tidak valid")
	}
	_ = u
	semester, err := strconv.Atoi(c.Query("semester", "0"))
	if err != nil || semester < 0 || semester > 14 {
		return helper.FailValidation(c, map[string]string{"semester": "Semester harus 1-14"})
	}
	search := strings.TrimSpace(c.Query("search"))
	available := strings.EqualFold(c.Query("available"), "true")
	rows, err := s.db.Query(c.Context(), `
        SELECT c.id,c.kode_mk,c.nama_mk,c.sks,c.semester,c.kuota,
            COUNT(e.id)::int AS terisi,
            GREATEST(c.kuota-COUNT(e.id)::int,0) AS sisa_kuota
        FROM courses c
        LEFT JOIN enrollments e ON e.course_id=c.id
        WHERE ($1=0 OR c.semester=$1)
        AND ($2='' OR c.kode_mk ILIKE '%'||$2||'%' OR c.nama_mk ILIKE '%'||$2||'%')
        GROUP BY c.id
        HAVING (NOT $3 OR COUNT(e.id)<c.kuota)
        ORDER BY c.semester,c.kode_mk`, semester, search, available)
	if err != nil {
		return helper.Fail(c, 500, "Gagal mengambil mata kuliah")
	}
	defer rows.Close()
	items := make([]fiber.Map, 0)
	for rows.Next() {
		var id, sks, sem, kuota, terisi, sisa int
		var kode, nama string
		if err := rows.Scan(&id, &kode, &nama, &sks, &sem, &kuota, &terisi, &sisa); err != nil {
			return helper.Fail(c, 500, "Gagal membaca mata kuliah")
		}
		items = append(items, fiber.Map{
			"id": id, "kode_mk": kode, "nama_mk": nama, "sks": sks, "semester": sem,
			"kuota": kuota, "terisi": terisi, "sisa_kuota": sisa,
		})
	}
	return helper.Success(c, 200, "Daftar mata kuliah", items)
}

func (s *SiakadService) CreateEnrollment(c *fiber.Ctx) error {
	u, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, 401, "Token tidak valid")
	}
	if u.Role != "mahasiswa" {
		return helper.Fail(c, 403, "Hanya mahasiswa yang dapat mengambil mata kuliah")
	}
	var req struct {
		CourseID int    `json:"course_id"`
		Tahun    string `json:"tahun_akademik"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, 400, "Format JSON tidak valid")
	}
	if req.CourseID < 1 || !yearPattern.MatchString(req.Tahun) {
		return helper.FailValidation(c, map[string]string{"course_id": "Course ID wajib positif", "tahun_akademik": "Format contoh: 2026/2027-Ganjil"})
	}
	ctx := c.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return helper.Fail(c, 500, "Gagal memulai transaksi")
	}
	defer tx.Rollback(ctx)

	var studentID int
	var ipk float64
	err = tx.QueryRow(ctx, `SELECT id,ipk_terakhir FROM students WHERE user_id=$1 AND deleted_at IS NULL FOR UPDATE`, u.ID).Scan(&studentID, &ipk)
	if err != nil {
		return helper.Fail(c, 403, "Akun tidak terhubung dengan mahasiswa aktif")
	}

	var kode, nama string
	var sks, kuota int
	err = tx.QueryRow(ctx, `SELECT kode_mk,nama_mk,sks,kuota FROM courses WHERE id=$1 FOR UPDATE`, req.CourseID).Scan(&kode, &nama, &sks, &kuota)
	if err == pgx.ErrNoRows {
		return helper.Fail(c, 404, "Mata kuliah tidak ditemukan")
	}
	if err != nil {
		return helper.Fail(c, 500, "Gagal mengambil mata kuliah")
	}

	var duplicate bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM enrollments WHERE student_id=$1 AND course_id=$2 AND tahun_akademik=$3)`, studentID, req.CourseID, req.Tahun).Scan(&duplicate)
	if err != nil {
		return helper.Fail(c, 500, "Gagal memeriksa KRS")
	}
	if duplicate {
		return helper.Fail(c, 409, "Mata kuliah ini sudah diambil pada tahun akademik tersebut")
	}

	var terisi int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM enrollments WHERE course_id=$1 AND tahun_akademik=$2`, req.CourseID, req.Tahun).Scan(&terisi)
	if err != nil {
		return helper.Fail(c, 500, "Gagal memeriksa kuota")
	}
	if terisi >= kuota {
		return helper.Fail(c, 422, "Kuota mata kuliah sudah penuh")
	}

	batas := 18
	if ipk >= 3 {
		batas = 24
	} else if ipk >= 2.5 {
		batas = 21
	}
	var total int
	err = tx.QueryRow(ctx, `
        SELECT COALESCE(SUM(c.sks),0)::int FROM enrollments e
        JOIN courses c ON c.id=e.course_id
        WHERE e.student_id=$1 AND e.tahun_akademik=$2`,
		studentID, req.Tahun).Scan(&total)
	if err != nil {
		return helper.Fail(c, 500, "Gagal menghitung SKS")
	}
	remaining := batas - total
	if sks > remaining {
		return helper.Fail(c, 422, fmt.Sprintf("SKS tidak mencukupi. Sisa SKS kamu %d, mata kuliah ini membutuhkan %d SKS.", remaining, sks))
	}

	var enrollmentID int
	err = tx.QueryRow(ctx, `INSERT INTO enrollments(student_id,course_id,tahun_akademik) VALUES($1,$2,$3) RETURNING id`, studentID, req.CourseID, req.Tahun).Scan(&enrollmentID)
	if err != nil {
		if isUniqueViolation(err) {
			return helper.Fail(c, 409, "Mata kuliah ini sudah diambil")
		}
		return helper.Fail(c, 500, "Gagal menyimpan KRS")
	}
	if err = tx.Commit(ctx); err != nil {
		return helper.Fail(c, 500, "Gagal menyimpan KRS")
	}
	return helper.Created(c, "KRS berhasil ditambahkan", fiber.Map{
		"id": enrollmentID, "student_id": studentID, "course_id": req.CourseID,
		"kode_mk": kode, "nama_mk": nama, "sks": sks, "tahun_akademik": req.Tahun,
	}, fmt.Sprintf("/api/v1/enrollments/%d", enrollmentID))
}

func (s *SiakadService) DeleteEnrollment(c *fiber.Ctx) error {
	u, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, 401, "Token tidak valid")
	}
	if u.Role != "mahasiswa" {
		return helper.Fail(c, 403, "Hanya mahasiswa yang dapat membatalkan KRS")
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return helper.Fail(c, 400, "ID enrollment tidak valid")
	}

	var studentID int
	err = s.db.QueryRow(c.Context(), `SELECT id FROM students WHERE user_id=$1 AND deleted_at IS NULL`, u.ID).Scan(&studentID)
	if err != nil {
		return helper.Fail(c, 403, "Akun tidak terhubung dengan mahasiswa aktif")
	}

	var ownerID int
	err = s.db.QueryRow(c.Context(), `SELECT student_id FROM enrollments WHERE id=$1`, id).Scan(&ownerID)
	if err == pgx.ErrNoRows {
		return helper.Fail(c, 404, "KRS tidak ditemukan")
	}
	if err != nil {
		return helper.Fail(c, 500, "Gagal mencari KRS")
	}
	if ownerID != studentID {
		return helper.Fail(c, 403, "Kamu tidak boleh menghapus KRS mahasiswa lain")
	}

	_, err = s.db.Exec(c.Context(), `DELETE FROM enrollments WHERE id=$1`, id)
	if err != nil {
		return helper.Fail(c, 500, "Gagal menghapus KRS")
	}
	return helper.Success(c, 200, "KRS berhasil dibatalkan", fiber.Map{"id": id})
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return err != nil && (asPgError(err, &pgErr) && pgErr.Code == "23505")
}

func asPgError(err error, target **pgconn.PgError) bool {
	for err != nil {
		if e, ok := err.(*pgconn.PgError); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = u.Unwrap()
	}
	return false
}
