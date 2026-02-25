// /data/gin-alarm-api/internal/ticketing/models/ticket_full_view.go
package models

import "time"

type TicketFullView struct {
	Header TicketHeader `json:"header"`
	Text   TicketText   `json:"text"`
}

type TicketLockView struct {
	Locked        bool       `json:"locked"`
	LockedBy      *string    `json:"locked_by,omitempty"`
	LockedAt      *time.Time `json:"locked_at,omitempty"`
	LockExpiresAt *time.Time `json:"lock_expires_at,omitempty"`
	LockReasonID  *int16     `json:"lock_reason_id,omitempty"`
}

type TicketOwnerView struct {
	UserID     *string    `json:"user_id,omitempty"`
	AssignedBy *string    `json:"assigned_by,omitempty"`
	AssignedAt *time.Time `json:"assigned_at,omitempty"`
}


