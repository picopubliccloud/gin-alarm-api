package repo

import (
	"context"
	"database/sql"
)

type UserCapabilities struct {
	CanViewRestricted  bool
	CanManageSLA       bool
	CanMergeTickets    bool
	CanAssignAny       bool
	CanDeclareIncident bool
}

type RBACRepo struct{ db *sql.DB }

func NewRBACRepo(db *sql.DB) *RBACRepo { return &RBACRepo{db: db} }

func (r *RBACRepo) GetUserActorType(ctx context.Context, externalSubject string) (string, string, error) {
	const q = `
SELECT user_id::text, actor_type::text
FROM ops.users
WHERE external_subject=$1 AND is_active=true;
`
	var uid, actor string
	if err := r.db.QueryRowContext(ctx, q, externalSubject).Scan(&uid, &actor); err != nil {
		return "", "", err
	}
	return uid, actor, nil
}

func (r *RBACRepo) GetUserCapabilities(ctx context.Context, userID string) (UserCapabilities, error) {
	const q = `
SELECT
  COALESCE(bool_or(ro.can_view_restricted), false),
  COALESCE(bool_or(ro.can_manage_sla), false),
  COALESCE(bool_or(ro.can_merge_tickets), false),
  COALESCE(bool_or(ro.can_assign_any), false),
  COALESCE(bool_or(ro.can_declare_incident), false)
FROM ops.user_project_memberships m
JOIN ops.roles ro ON ro.role_id = m.role_id
WHERE m.user_id = $1::uuid;
`
	var c UserCapabilities
	err := r.db.QueryRowContext(ctx, q, userID).Scan(
		&c.CanViewRestricted,
		&c.CanManageSLA,
		&c.CanMergeTickets,
		&c.CanAssignAny,
		&c.CanDeclareIncident,
	)
	return c, err
}

func (r *RBACRepo) GetUserProjectScope(ctx context.Context, userID string) ([]string, error) {
	const q = `
SELECT DISTINCT project_id
FROM ops.user_project_memberships
WHERE user_id=$1::uuid;
`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
