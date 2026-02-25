package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
)

type UpdateRepo struct{ db *sql.DB }

func NewUpdateRepo(db *sql.DB) *UpdateRepo { return &UpdateRepo{db: db} }

func (r *UpdateRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return beginTx(ctx, r.db)
}

func (r *UpdateRepo) Insert(
	ctx context.Context,
	tx *sql.Tx,
	ticketID string,
	req *models.AddUpdateRequest,
	userID string,
	actor models.ActorType,
) error {

	var structured any
	if req.Structured != nil {
		b, err := json.Marshal(req.Structured)
		if err != nil {
			return err
		}
		structured = b
	}

	const q = `
INSERT INTO ops.ticket_updates
(ticket_id, created_by, created_by_actor, update_type, visibility, body, structured)
VALUES ($1,$2,$3,$4,$5,$6,$7);
`
	_, err := tx.ExecContext(
		ctx, q,
		ticketID,
		userID,
		string(actor),
		string(req.UpdateType),
		string(req.Visibility),
		req.Body,
		structured,
	)
	return err
}


func (r *UpdateRepo) ListByTicket(
	ctx context.Context,
	ticketID string,
	maxVis models.TicketVisibility,
	limit int,
) ([]models.TicketUpdate, error) {

	if limit <= 0 || limit > 200 {
		limit = 100
	}

	conds := []string{"u.ticket_id = $1::uuid"}
	args := []any{ticketID}
	argn := 2

	// Visibility filter
	if maxVis == models.VisibilityPublic {
		conds = append(conds, fmt.Sprintf("u.visibility = $%d", argn))
		args = append(args, string(models.VisibilityPublic))
		argn++
	} else if maxVis == models.VisibilityInternal {
		conds = append(conds, fmt.Sprintf("u.visibility IN ($%d,$%d)", argn, argn+1))
		args = append(args, string(models.VisibilityPublic), string(models.VisibilityInternal))
		argn += 2
	}

	q := fmt.Sprintf(`
SELECT
  u.update_id::text,
  u.ticket_id::text,
  COALESCE(usr.display_name, u.created_by::text),
  u.created_by_actor::text,
  u.update_type::text,
  u.visibility::text,
  u.body,
  u.structured,
  u.created_at
FROM ops.ticket_updates u
LEFT JOIN ops.users usr ON usr.user_id = u.created_by
WHERE %s
ORDER BY u.created_at DESC
LIMIT %d;
`, strings.Join(conds, " AND "), limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.TicketUpdate, 0, limit)

	for rows.Next() {
		var u models.TicketUpdate

		var actor, updType, vis string
		var body sql.NullString
		var structured []byte

		if err := rows.Scan(
			&u.UpdateID,
			&u.TicketID,
			&u.CreatedBy,
			&actor,
			&updType,
			&vis,
			&body,
			&structured,
			&u.CreatedAt,
		); err != nil {
			return nil, err
		}

		u.Actor = models.ActorType(actor)
		u.UpdateType = models.UpdateType(updType)
		u.Visibility = models.TicketVisibility(vis)

		if body.Valid {
			s := body.String
			u.Body = &s
		}

		if len(structured) > 0 {
			u.Structured = json.RawMessage(structured)
		}

		out = append(out, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

