package repo

import (
	"context"
	"database/sql"
	"time"
)

type SLARepo struct{ db *sql.DB }

func NewSLARepo(db *sql.DB) *SLARepo { return &SLARepo{db: db} }

// Start creates ticket_sla row from sla_policies for that ticket (if policy exists).
func (r *SLARepo) Start(ctx context.Context, tx *sql.Tx, ticketID string) error {
	// find tier + ticket_type + severity
	const qPolicy = `
SELECT p.sla_policy_id::text, p.first_response_minutes, p.resolve_minutes
FROM ops.tickets_header h
JOIN ops.customers c ON c.customer_id = h.customer_id
JOIN ops.customer_tiers t ON t.tier_id = c.tier_id
JOIN ops.sla_policies p ON p.tier_id = t.tier_id
                      AND p.ticket_type = h.ticket_type
                      AND p.severity = h.severity
WHERE h.ticket_id = $1::uuid
LIMIT 1;
`
	var policyID string
	var frMin, resMin int
	err := tx.QueryRowContext(ctx, qPolicy, ticketID).Scan(&policyID, &frMin, &resMin)
	if err == sql.ErrNoRows {
		return nil // no policy configured yet
	}
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	firstDue := now.Add(time.Duration(frMin) * time.Minute)
	resolveDue := now.Add(time.Duration(resMin) * time.Minute)

	const qInsert = `
INSERT INTO ops.ticket_sla (ticket_id, sla_policy_id, first_response_due_at, resolve_due_at)
VALUES ($1::uuid, $2::uuid, $3, $4)
ON CONFLICT (ticket_id) DO NOTHING;
`
	_, err = tx.ExecContext(ctx, qInsert, ticketID, policyID, firstDue, resolveDue)
	return err
}
