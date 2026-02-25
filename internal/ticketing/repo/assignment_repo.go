package repo

import (
	"context"
	"database/sql"
  "strings"
)

type AssignmentRepo struct{ db *sql.DB }

func NewAssignmentRepo(db *sql.DB) *AssignmentRepo { return &AssignmentRepo{db: db} }

func (r *AssignmentRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return beginTx(ctx, r.db)
}

// func (r *AssignmentRepo) AssignOwner(
// 	ctx context.Context,
// 	tx *sql.Tx,
// 	ticketID string,
// 	userID string,
// 	assignedBy string,
// ) error {
// 	const q = `
// INSERT INTO ops.ticket_assignments
// (ticket_id, user_id, assignment_role, assigned_by, assigned_at)
// VALUES ($1::uuid, $2::uuid, 'OWNER', $3::uuid, now());
// `
// 	_, err := tx.ExecContext(ctx, q, ticketID, userID, assignedBy)
// 	return err
// }
// /internal/ticketing/repo/assignment_repo.go

func (r *AssignmentRepo) AssignOwner(
	ctx context.Context,
	tx *sql.Tx,
	ticketID string,
	userID string,      // target owner
	assignedBy string,  // actor
) error {

	// 1) End any existing active OWNER assignment for this ticket
	// Active = unassigned_at IS NULL (per your unique index)
	_, err := tx.ExecContext(ctx, `
UPDATE ops.ticket_assignments
SET
  unassigned_at = NOW(),
  unassigned_by = $2::uuid,
  unassign_reason_id = 2,          -- optional default reason (pick what you like)
  unassign_note = 'reassigned'     -- optional
WHERE ticket_id = $1::uuid
  AND assignment_role = 'OWNER'::ops.assignment_role
  AND unassigned_at IS NULL
`, ticketID, assignedBy)
	if err != nil {
		return err
	}

	// 2) Insert the new active OWNER assignment
	_, err = tx.ExecContext(ctx, `
INSERT INTO ops.ticket_assignments
(ticket_id, user_id, assignment_role, assigned_by, assigned_at)
VALUES ($1::uuid, $2::uuid, 'OWNER'::ops.assignment_role, $3::uuid, NOW())
`, ticketID, userID, assignedBy)

	return err
}






// func (r *AssignmentRepo) Unassign(
// 	ctx context.Context,
// 	tx *sql.Tx,
// 	ticketID string,
// 	by string,
// 	reason *string,
// ) error {
// 	const q = `
// UPDATE ops.ticket_assignments
// SET unassigned_at = now(),
//     unassign_note = $3
// WHERE ticket_id = $1::uuid
//   AND assignment_role = 'OWNER'
//   AND unassigned_at IS NULL;
// `
// 	fmt.Println("Unassign query: ",q,ticketID, by, reason)
// 	_, err := tx.ExecContext(ctx, q, ticketID, by, reason)
// 	return err
// }

func (r *AssignmentRepo) Unassign(
    ctx context.Context,
    tx *sql.Tx,
    ticketID string,
    unassignedBy string,
    unassignReasonID *int16,
    note *string,
) error {
    const q = `
UPDATE ops.ticket_assignments
SET
  unassigned_at      = now(),
  unassigned_by      = $2::uuid,
  unassign_reason_id = $3::smallint,
  unassign_note      = $4
WHERE ticket_id = $1::uuid
  AND assignment_role = 'OWNER'
  AND unassigned_at IS NULL;
`
    _, err := tx.ExecContext(ctx, q, ticketID, unassignedBy, unassignReasonID, note)
    return err
}


func (r *AssignmentRepo) GetUserDisplayName(
	ctx context.Context,
	tx *sql.Tx,
	userID string,
) (string, error) {

	var name sql.NullString

	err := tx.QueryRowContext(ctx, `
SELECT display_name
FROM ops.users
WHERE user_id = $1::uuid
`, userID).Scan(&name)

	if err != nil {
		return "", err
	}

	if name.Valid && strings.TrimSpace(name.String) != "" {
		return strings.TrimSpace(name.String), nil
	}

	return userID, nil // fallback
}

func (r *AssignmentRepo) GetUnassignReasonName(ctx context.Context, tx *sql.Tx, reasonID int16) (string, error) {
	var name string
	err := tx.QueryRowContext(ctx, `
SELECT name
FROM ops.unassign_reasons
WHERE unassign_reason_id = $1
`, reasonID).Scan(&name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}
