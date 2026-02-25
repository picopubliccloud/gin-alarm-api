package models

import "time"

type LockTicketRequest struct {
	ExpiresInSeconds int   `json:"expires_in_seconds" binding:"required"`
	LockReasonID     *int16 `json:"lock_reason_id"`
}

type TicketLock struct {
	TicketID       string    `json:"ticket_id"`
	LockedBy       string    `json:"locked_by"`
	LockedAt       time.Time `json:"locked_at"`
	LockExpiresAt  time.Time `json:"lock_expires_at"`
	LockReasonID   *int16     `json:"lock_reason_id,omitempty"`
}
