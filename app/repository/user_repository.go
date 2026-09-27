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
	argNumber := 1

	if q.Search != "" {
		conditions = append(
			conditions,
			fmt.Sprintf("(username ILIKE $%d OR email ILIKE $%d)", argNumber, argNumber),
		)
		args = append(args, "%"+q.Search+"%")
		argNumber++
	}

	if q.IsActive != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("is_active = $%d", argNumber),
		)
		args = append(args, *q.IsActive)
		argNumber++
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Whitelist kolom untuk mencegah SQL injection pada ORDER BY.
	sortColumns := map[string]string{
		"id":         "id",
		"username":   "username",
		"email":      "email",
		"is_active":  "is_active",
		"created_at": "created_at",
	}

	sortColumn, ok := sortColumns[q.Sort]
	if !ok {
		sortColumn = "id"
	}

	order := "ASC"
	if strings.ToLower(q.Order) == "desc" {
		order = "DESC"
	}

	// Hitung total data sesuai filter.
	var total int

	countQuery := "SELECT COUNT(*) FROM users" + where

	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Tambahkan LIMIT dan OFFSET.
	queryArgs := append([]any{}, args...)

	limitPlaceholder := argNumber
	offsetPlaceholder := argNumber + 1

	queryArgs = append(
		queryArgs,
		q.Limit,
		q.Offset(),
	)

	query := fmt.Sprintf(`
		SELECT id, username, email, password, is_active, created_at
		FROM users
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`,
		where,
		sortColumn,
		order,
		limitPlaceholder,
		offsetPlaceholder,
	)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := []model.User{}

	for rows.Next() {
		var u model.User

		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.Password,
			&u.IsActive,
			&u.CreatedAt,
		)

		if err != nil {
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
		SELECT id, username, email, password, is_active, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.Password,
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
		INSERT INTO users
			(username, email, password, is_active)
		VALUES
			($1, $2, $3, $4)
		RETURNING id, created_at
	`,
		u.Username,
		u.Email,
		u.Password,
		u.IsActive,
	).Scan(
		&u.ID,
		&u.CreatedAt,
	)
}

func (r *UserRepository) Update(
	ctx context.Context,
	u *model.User,
) error {

	cmd, err := r.db.Exec(ctx, `
		UPDATE users
		SET username = $1,
			email = $2,
			is_active = $3
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
		DELETE FROM users
		WHERE id = $1
	`, id)

	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil

}

type UserRepositoryInterface interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.User, int, error)
	FindByID(ctx context.Context, id int) (*model.User, error)
	Create(ctx context.Context, u *model.User) error
	Update(ctx context.Context, u *model.User) error
	Delete(ctx context.Context, id int) error
}
