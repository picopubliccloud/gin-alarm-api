package models

type User struct {
	UserID          string
	ExternalSubject string
	Email           string
	DisplayName     string
	ActorType       ActorType
	IsActive        bool
}
