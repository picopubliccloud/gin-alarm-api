package models

import "time"

type TicketUpdate struct {
	UpdateID   string           `json:"update_id"`
	TicketID   string           `json:"ticket_id"`
	CreatedBy  string           `json:"created_by"`
	Actor      ActorType        `json:"created_by_actor"`
	UpdateType UpdateType       `json:"update_type"`
	Visibility TicketVisibility `json:"visibility"`
	Body       *string          `json:"body"`
	Structured any              `json:"structured,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}

type AddUpdateRequest struct {
	Visibility TicketVisibility `json:"visibility" binding:"required"`
	UpdateType UpdateType       `json:"update_type" binding:"required"`
	Body       *string          `json:"body"`
	Structured any              `json:"structured,omitempty"`
}
