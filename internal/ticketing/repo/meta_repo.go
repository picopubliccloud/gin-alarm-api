// /internal/ticketing/repo/meta_repo.go
package repo

import (
	"context"
	"database/sql"
	"strings"
	"fmt"

	"github.com/google/uuid"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
)

type MetaRepo struct {
	DB *sql.DB
}

func NewMetaRepo(db *sql.DB) *MetaRepo {
	return &MetaRepo{DB: db}
}

func (r *MetaRepo) ListCustomers(ctx context.Context, onlyActive bool) ([]models.CustomerOption, error) {
	q := `
SELECT customer_id, name
FROM ops.customers
`
	args := []any{}
	if onlyActive {
		q += ` WHERE is_active = true`
	}
	q += ` ORDER BY name ASC`

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.CustomerOption, 0, 64)
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out = append(out, models.CustomerOption{ID: id, Name: strings.TrimSpace(name)})
	}
	return out, rows.Err()
}

func (r *MetaRepo) ListServices(ctx context.Context, onlyActive bool) ([]models.ServiceOption, error) {
	q := `
SELECT
  sc.service_id,
  cat.name  AS category_name,
  p.name    AS product_name,
  st.name   AS type_name
FROM ops.service_catalog sc
JOIN ops.service_categories cat ON cat.category_id = sc.category_id
JOIN ops.products p            ON p.product_id = sc.product_id
LEFT JOIN ops.service_types st ON st.type_id = sc.type_id
`
	args := []any{}
	if onlyActive {
		q += ` WHERE sc.is_active = true`
	}
	q += ` ORDER BY cat.name ASC, p.name ASC, st.name ASC NULLS LAST`

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.ServiceOption, 0, 256)
	for rows.Next() {
		var id uuid.UUID
		var cat, prod string
		var typ sql.NullString

		if err := rows.Scan(&id, &cat, &prod, &typ); err != nil {
			return nil, err
		}

		label := strings.TrimSpace(cat) + " / " + strings.TrimSpace(prod)
		if typ.Valid && strings.TrimSpace(typ.String) != "" {
			label += " / " + strings.TrimSpace(typ.String)
		}

		out = append(out, models.ServiceOption{
			ID:    id,
			Label: label,
		})
	}
	return out, rows.Err()
}

func (r *MetaRepo) GetDefaultProjectIDByCustomer(ctx context.Context, customerID uuid.UUID) (string, error) {
	q := `
SELECT project_id
FROM ops.projects
WHERE customer_id = $1
  AND is_active = true
ORDER BY project_name ASC
LIMIT 1
`
	var projectID string
	err := r.DB.QueryRowContext(ctx, q, customerID).Scan(&projectID)
	if err == nil {
		return strings.TrimSpace(projectID), nil
	}

	if err == sql.ErrNoRows {
		q2 := `
SELECT project_id
FROM ops.projects
WHERE customer_id = $1
ORDER BY project_name ASC
LIMIT 1
`
		err2 := r.DB.QueryRowContext(ctx, q2, customerID).Scan(&projectID)
		if err2 != nil {
			return "", err2
		}
		return strings.TrimSpace(projectID), nil
	}

	return "", err
}

func (r *MetaRepo) ListUsers(ctx context.Context, q string, active bool, onlyOps bool, limit int) ([]models.UserOption, error) {
	if limit <= 0 || limit > 200 {
		limit = 80
	}

	conds := []string{"1=1"}
	args := []any{}
	argn := 1

	if active {
		conds = append(conds, fmt.Sprintf("u.is_active = $%d", argn))
		args = append(args, true)
		argn++
	}

	if onlyOps {
		conds = append(conds, fmt.Sprintf("u.actor_type <> $%d", argn))
		args = append(args, "CUSTOMER")
		argn++
	}

	if q != "" {
		conds = append(conds, fmt.Sprintf("(u.display_name ILIKE $%d OR u.email::text ILIKE $%d)", argn, argn))
		args = append(args, "%"+q+"%")
		argn++
	}

	// limit is last param
	args = append(args, limit)

	sqlq := fmt.Sprintf(`
SELECT
  u.user_id::text,
  u.display_name,
  u.email::text,
  u.actor_type::text
FROM ops.users u
WHERE %s
ORDER BY u.display_name NULLS LAST, u.email NULLS LAST
LIMIT $%d
`, strings.Join(conds, " AND "), argn)

	rows, err := r.DB.QueryContext(ctx, sqlq, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.UserOption, 0, 80)
	for rows.Next() {
		var x models.UserOption
		var email *string
		if err := rows.Scan(&x.UserID, &x.DisplayName, &email, &x.ActorType); err != nil {
			return nil, err
		}
		x.Email = email
		out = append(out, x)
	}
	return out, rows.Err()
}