package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"latihan-fiber/app/model"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindAll(
	ctx context.Context,
	q model.ListQuery,
) ([]model.User, int, error) {
	conditions := []string{}
	args := []any{}

	// Pencarian user.
	if q.Search != "" {
		args = append(args, "%"+q.Search+"%")
		conditions = append(conditions, fmt.Sprintf(
			"(username ILIKE $%d OR email ILIKE $%d)",
			len(args), len(args),
		))
	}

	// Filter status aktif.
	if q.IsActive != nil {
		args = append(args, *q.IsActive)
		conditions = append(conditions, fmt.Sprintf(
			"is_active = $%d", len(args),
		))
	}

	// Hitung total berdasarkan filter, sebelum cursor diterapkan.
	countWhere := ""
	if len(conditions) > 0 {
		countWhere = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countArgs := append([]any{}, args...)
	err := r.db.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM users"+countWhere,
		countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Cursor pagination menggunakan (created_at, id).
	if q.CursorTime != nil && q.CursorID > 0 {
		args = append(args, *q.CursorTime, q.CursorID)
		conditions = append(conditions, fmt.Sprintf(
			"(created_at, id) > ($%d, $%d)",
			len(args)-1, len(args),
		))
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Ambil satu data ekstra untuk mendeteksi halaman berikutnya.
	limit := q.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	args = append(args, limit+1)

	query := fmt.Sprintf(`
		SELECT id, username, email, password, role, is_active, created_at
		FROM users
		%s
		ORDER BY created_at ASC, id ASC
		LIMIT $%d
	`, where, len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := []model.User{}
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.Password,
			&u.Role,
			&u.IsActive,
			&u.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) FindByID(
	ctx context.Context,
	id int,
) (*model.User, error) {
	var u model.User

	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password, role, is_active, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepository) FindByUsername(
	ctx context.Context,
	username string,
) (*model.User, error) {
	var u model.User

	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password, role, is_active, created_at
		FROM users
		WHERE username = $1
	`, username).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepository) Create(
	ctx context.Context,
	u *model.User,
) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO users (username, email, password, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`,
		u.Username,
		u.Email,
		u.Password,
		u.IsActive,
	).Scan(&u.ID, &u.CreatedAt)
}

func (r *UserRepository) Update(
	ctx context.Context,
	u *model.User,
) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE users
		SET username = $1, email = $2, is_active = $3
		WHERE id = $4
	`,
		u.Username,
		u.Email,
		u.IsActive,
		u.ID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *UserRepository) Delete(
	ctx context.Context,
	id int,
) error {
	cmd, err := r.db.Exec(ctx, `
		DELETE FROM users WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *UserRepository) SetRole(
	ctx context.Context,
	id int,
	role string,
) error {
	cmd, err := r.db.Exec(
		ctx,
		`UPDATE users SET role = $1 WHERE id = $2`,
		role,
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

type UserRepositoryInterface interface {
	SetRole(ctx context.Context, id int, role string) error
	FindAll(ctx context.Context, q model.ListQuery) ([]model.User, int, error)
	FindByID(ctx context.Context, id int) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	Create(ctx context.Context, u *model.User) error
	Update(ctx context.Context, u *model.User) error
	Delete(ctx context.Context, id int) error
}
