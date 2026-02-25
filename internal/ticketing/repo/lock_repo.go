package repo

import (
	"context"
	"database/sql"
	"time"
)

type LockRepo struct{ db *sql.DB }

func NewLockRepo(db *sql.DB) *LockRepo { return &LockRepo{db: db} }

// s.LockRepo.TryLock(ctx, tx, ticketID, userID, LockReasonID, ttl)
// ticketID string, userID string, LockReasonID int16, ttl int64
func (r *LockRepo) TryLock(
	ctx context.Context,
	tx *sql.Tx,
	ticketID string,
	lockedBy string,
	LockReasonID int16,
	ttl int64,
) error {

	expires := time.Now().UTC().Add(time.Duration(ttl) * time.Second)

	const q = `
INSERT INTO ops.ticket_locks (ticket_id, locked_by, locked_at, lock_expires_at, lock_reason_id)
VALUES ($1::uuid,$2::uuid,now(),$3, $4)
ON CONFLICT (ticket_id) DO UPDATE
SET locked_by=EXCLUDED.locked_by,
    locked_at=now(),
    lock_expires_at=EXCLUDED.lock_expires_at,
	lock_reason_id=EXCLUDED.lock_reason_id
WHERE ops.ticket_locks.lock_expires_at < now();
`

	exec := r.db.ExecContext
	if tx != nil {
		exec = tx.ExecContext
	}

	res, err := exec(ctx, q, ticketID, lockedBy, expires, LockReasonID)
	if err != nil {
		return err
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *LockRepo) Release(ctx context.Context, ticketID, userID string) error {
	const q = `
DELETE FROM ops.ticket_locks
WHERE ticket_id=$1::uuid AND locked_by=$2::uuid;
`
	_, err := r.db.ExecContext(ctx, q, ticketID, userID)
	return err
}
