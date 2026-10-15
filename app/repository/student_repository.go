package repository

import (
    "context"

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

func (r *StudentRepository) FindAll(ctx context.Context) ([]model.Student, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, nim, nama, prodi, owner_id, created_at
        FROM students ORDER BY id
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    students := []model.Student{}
    for rows.Next() {
        var s model.Student
        if err := rows.Scan(&s.ID, &s.NIM, &s.Nama, &s.Prodi, &s.OwnerID, &s.CreatedAt); err != nil {
            return nil, err
        }
        students = append(students, s)
    }
    return students, rows.Err()
}

func (r *StudentRepository) FindByID(ctx context.Context, id int) (*model.Student, error) {
    var s model.Student
    err := r.db.QueryRow(ctx, `
        SELECT id, nim, nama, prodi, owner_id, created_at
        FROM students WHERE id = $1
    `, id).Scan(&s.ID, &s.NIM, &s.Nama, &s.Prodi, &s.OwnerID, &s.CreatedAt)
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
    cmd, err := r.db.Exec(ctx, `DELETE FROM students WHERE id = $1`, id)
    if err != nil {
        return err
    }
    if cmd.RowsAffected() == 0 {
        return pgx.ErrNoRows
    }
    return nil
}
