package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type OutboxRepo struct{ db *sql.DB }

func NewOutboxRepo(db *sql.DB) *OutboxRepo { return &OutboxRepo{db: db} }

func (r *OutboxRepo) Emit(
	ctx context.Context,
	tx *sql.Tx,
	aggregateType string,
	aggregateID string,
	eventType string,
	payload any,
) error {

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	const q = `
INSERT INTO ops.outbox_events
(aggregate_type, aggregate_id, event_type, payload, created_at)
VALUES ($1,$2::uuid,$3,$4::jsonb,now());
`
	_, err = tx.ExecContext(ctx, q, aggregateType, aggregateID, eventType, b)
	return err
}

func (r *OutboxRepo) Process(ctx context.Context, batch int) (int, error) {
	if batch <= 0 {
		batch = 100
	}

	const q = `
WITH picked AS (
  SELECT event_id
  FROM ops.outbox_events
  WHERE processed_at IS NULL
  ORDER BY created_at
  LIMIT $1
  FOR UPDATE SKIP LOCKED
)
UPDATE ops.outbox_events
SET processed_at=$2
FROM picked
WHERE ops.outbox_events.event_id=picked.event_id;
`
	res, err := r.db.ExecContext(ctx, q, batch, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
