package models

type RBACContext struct {
	AllowedProjectIDs []string
	MaxVisibility     TicketVisibility
	IsOps             bool
	// Useful for UI / feature toggles
	CanViewRestricted  bool
	CanManageSLA       bool
	CanMergeTickets    bool
	CanAssignAny       bool
	CanDeclareIncident bool
}
