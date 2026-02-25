package models

type SearchRequest struct {
	Query string
	Limit int
}

type SearchResult struct {
	TicketID     string
	TicketNumber int64
	Title        string
	Score        float64
}
