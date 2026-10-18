package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"latihan-fiber/app/model"
)

type StudentRepository struct {
	db *pgxpool.Pool
}

func NewStudentRepository(db *pgxpool.Pool) *StudentRepository {
	return &StudentRepository{db: db}
}

func (r *StudentRepository) Create(ctx context.Context, s *model.Student) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO students (nim, nama, prodi, owner_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, s.NIM, s.Nama, s.Prodi, s.OwnerID).Scan(&s.ID, &s.CreatedAt)
}

func (r *StudentRepository) FindAll(
	ctx context.Context,
	q model.ListQuery,
) ([]model.Student, error) {
	var conditions []string
	var args []any

	if q.Search != "" {
		args = append(args, "%"+strings.TrimSpace(q.Search)+"%")
		conditions = append(
			conditions,
			fmt.Sprintf("(nim ILIKE $%d OR nama ILIKE $%d OR prodi ILIKE $%d)",
				len(args), len(args), len(args)),
		)
	}

	if q.CursorTime != nil && q.CursorID > 0 {
		args = append(args, *q.CursorTime, q.CursorID)
		conditions = append(
			conditions,
			fmt.Sprintf("(created_at, id) > ($%d, $%d)",
				len(args)-1, len(args)),
		)
	}

	query := `
		SELECT id, nim, nama, prodi, owner_id, created_at
		FROM students
	`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	limit := q.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	args = append(args, limit+1)
	query += fmt.Sprintf(
		" ORDER BY created_at ASC, id ASC LIMIT $%d",
		len(args),
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []model.Student{}

	for rows.Next() {
		var s model.Student
		if err := rows.Scan(
			&s.ID,
			&s.NIM,
			&s.Nama,
			&s.Prodi,
			&s.OwnerID,
			&s.CreatedAt,
		); err != nil {
			return nil, err
		}
		students = append(students, s)
	}

	return students, rows.Err()
}

func (r *StudentRepository) FindByID(
	ctx context.Context,
	id int,
) (*model.Student, error) {
	var s model.Student

	err := r.db.QueryRow(ctx, `
		SELECT id, nim, nama, prodi, owner_id, created_at
		FROM students WHERE id = $1
	`, id).Scan(
		&s.ID, &s.NIM, &s.Nama, &s.Prodi, &s.OwnerID, &s.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *StudentRepository) Update(ctx context.Context, s *model.Student) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE students SET nim = $1, nama = $2, prodi = $3
		WHERE id = $4
	`, s.NIM, s.Nama, s.Prodi, s.ID)

	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *StudentRepository) Delete(ctx context.Context, id int) error {
	cmd, err := r.db.Exec(
		ctx,
		`DELETE FROM students WHERE id = $1`,
		id,
	)

	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
