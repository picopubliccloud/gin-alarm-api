package models

type TicketFilter struct {
	Status     TicketStatus
	Severity   TicketSeverity
	OnlyPool   bool
	Cursor     int64
	Limit      int
}
type TicketsSummaryResponse struct {
	Total      int64            `json:"total"`
	ByStatus   map[string]int64 `json:"by_status,omitempty"`
	BySeverity map[string]int64 `json:"by_severity,omitempty"`
}