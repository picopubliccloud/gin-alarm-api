package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
)

type ClosureRepo struct{ db *sql.DB }

func NewClosureRepo(db *sql.DB) *ClosureRepo { return &ClosureRepo{db: db} }

func (r *ClosureRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return beginTx(ctx, r.db)
}

func (r *ClosureRepo) Upsert(
	ctx context.Context,
	tx *sql.Tx,
	ticketID string,
	req *models.CloseTicketRequest,
	closedBy string,
) error {

	now := time.Now().UTC()

	const q = `
INSERT INTO ops.ticket_closure_summaries (
  ticket_id, fix_headline, symptoms, root_cause,
  fix_applied, verification_steps, prevention,
  resolution_code, closed_by, closed_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (ticket_id) DO UPDATE SET
  fix_headline=EXCLUDED.fix_headline,
  symptoms=EXCLUDED.symptoms,
  root_cause=EXCLUDED.root_cause,
  fix_applied=EXCLUDED.fix_applied,
  verification_steps=EXCLUDED.verification_steps,
  prevention=EXCLUDED.prevention,
  resolution_code=EXCLUDED.resolution_code,
  closed_by=EXCLUDED.closed_by,
  closed_at=EXCLUDED.closed_at;
`
	_, err := tx.ExecContext(
		ctx, q,
		ticketID,
		req.FixHeadline,
		req.Symptoms,
		req.RootCause,
		req.FixApplied,
		req.VerificationSteps,
		req.Prevention,
		string(req.ResolutionCode),
		closedBy,
		now,
	)
	return err
}
