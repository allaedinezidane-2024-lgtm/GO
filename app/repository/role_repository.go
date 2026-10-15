package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) PermissionsByRole(ctx context.Context) (map[string][]string, error) {
	rows, err := r.db.Query(ctx, `
        SELECT r.name, p.name
        FROM roles r
        LEFT JOIN role_permissions rp ON rp.role_name = r.name
        LEFT JOIN permissions p ON p.name = rp.permission_name
        ORDER BY r.name, p.name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mapping := make(map[string][]string)

	for rows.Next() {
		var role string
		var permission *string

		if err := rows.Scan(&role, &permission); err != nil {
			return nil, err
		}

		if _, exists := mapping[role]; !exists {
			mapping[role] = []string{}
		}
		if permission != nil {
			mapping[role] = append(mapping[role], *permission)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mapping, nil
}
