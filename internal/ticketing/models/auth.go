package models

type AuthContext struct {
	Sub   string
	Email string
	Name  string
	Roles []string
}
