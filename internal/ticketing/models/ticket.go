package models

import "time"

type TicketListItem struct {
	TicketID     string          `json:"ticket_id"`
	TicketNumber int64           `json:"ticket_number"`
	ProjectID    string          `json:"project_id"`
	ProjectName   *string   `json:"project_name,omitempty"`
	CustomerID   string          `json:"customer_id"`
	ServiceID    string          `json:"service_id"`
	TicketType   TicketType      `json:"ticket_type"`
	Status       TicketStatus    `json:"status"`
	Severity     TicketSeverity  `json:"severity"`
	Priority     int             `json:"priority_score"`
	Visibility   TicketVisibility `json:"visibility"`
	ComponentID  *int16          `json:"component_id,omitempty"`
	IsKnownIssue bool            `json:"is_known_issue"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	HasOwner bool `json:"has_owner"`
	IsLocked bool `json:"is_locked"`

	OwnerUserID      *string    `json:"owner_user_id,omitempty"`
    OwnerDisplayName *string    `json:"owner_display_name,omitempty"`
    OwnerAssignedAt  *time.Time `json:"owner_assigned_at,omitempty"`

	Title string `json:"title"`
}



type TicketDetail struct {
	Header    TicketHeader `json:"header"`
	Text      TicketText   `json:"text"`
	Resources []any        `json:"resources"`
	Updates []TicketUpdate  `json:"updates,omitempty"`

	// computed views (single “current” owner/lock)
	Owner *TicketOwnerView `json:"owner,omitempty"`
	Lock  *TicketLockView  `json:"lock,omitempty"`
}

type TicketHeader struct {
	TicketID     string
	TicketNumber int64
	ProjectID    string
	CustomerID   string
	ServiceID    string
	TicketType   TicketType
	Status       TicketStatus
	Severity     TicketSeverity
	Priority     int
	Visibility   TicketVisibility
	ComponentID  *int16
	IsKnownIssue bool

	CreatedBy       string
	CreatedByActor  ActorType

	MasterTicketID *string
	ProblemID      *string

	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  *time.Time
	ClosedBy  *string

	ResolutionCode *ResolutionCode
}

type TicketText struct {
	TicketID        string
	Title           string
	Description     *string
	ImpactSummary   *string
	SuspectedCause  *string
	CreatedAt       time.Time
}

// CreateTicketRequest represents ticket creation payload from API
type CreateTicketRequest struct {
	ProjectID string `json:"project_id"`
	CustomerID     string           `json:"customer_id" binding:"required"`
	ServiceID      string           `json:"service_id" binding:"required"`

	TicketType     TicketType       `json:"ticket_type" binding:"required"`
	Severity       TicketSeverity   `json:"severity" binding:"required"`
	Visibility     TicketVisibility `json:"visibility" binding:"required"`

	ComponentID    *int16           `json:"component_id"`

	Title          string           `json:"title" binding:"required"`
	Description    *string          `json:"description"`
	ImpactSummary  *string          `json:"impact_summary"`
	SuspectedCause *string          `json:"suspected_cause"`
}


// ToModels converts API request → persistence models
func (r *CreateTicketRequest) ToModels(createdBy string, actor ActorType) (*TicketHeader, *TicketText) {
	h := &TicketHeader{
		ProjectID:      r.ProjectID,
		CustomerID:     r.CustomerID,
		ServiceID:      r.ServiceID,
		TicketType:     r.TicketType,
		Severity:       r.Severity,
		Visibility:     r.Visibility,
		ComponentID:    r.ComponentID,
		CreatedBy:      createdBy,
		CreatedByActor: actor,
	}

	t := &TicketText{
		Title:          r.Title,
		Description:    r.Description,
		ImpactSummary:  r.ImpactSummary,
		SuspectedCause: r.SuspectedCause,
	}

	return h, t
}

type CursorPage struct {
	Items      any    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}


// summary
type TicketsSummary struct {
	Total     int64            `json:"total"`
	ByStatus  map[string]int64 `json:"by_status"`
	BySeverity map[string]int64 `json:"by_severity"`
}

type SummaryFilter struct {
	Pool         bool
	Status       string
	Severity     string
	IncludeClosed bool
	AllowedProjectIDs []string
	MaxVisibility string // "PUBLIC" or "INTERNAL" 
	SubjectUserID string // if need it for "my tickets" mode later
}