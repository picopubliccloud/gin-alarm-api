package models

import "github.com/google/uuid"

type CustomerOption struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ServiceOption struct {
	ID    uuid.UUID `json:"id"`    // ops.service_catalog.service_id
	Label string    `json:"label"` // "Category / Product / Type"
}

type UserOption struct {
	UserID      string  `json:"user_id"`
	DisplayName *string `json:"display_name"`
	Email       *string `json:"email"`
	ActorType   string  `json:"actor_type"`
}
