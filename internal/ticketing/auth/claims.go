// internal/ticketing/auth/claims.go
package auth

import "github.com/golang-jwt/jwt/v5"

type RealmAccess struct {
	Roles []string `json:"roles"`
}

type ClientRoles struct {
	Roles []string `json:"roles"`
}

type KCClaims struct {
	jwt.RegisteredClaims

	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`

	RealmAccess    RealmAccess            `json:"realm_access"`
	ResourceAccess map[string]ClientRoles `json:"resource_access"`
}
