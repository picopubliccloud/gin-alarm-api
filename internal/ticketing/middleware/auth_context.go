package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/auth"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
)

const (
	CtxAuth = "ticket_auth"
)

func AuthContext(v *auth.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Let OPTIONS pass. CORS middleware should answer it.
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		authz := c.GetHeader("Authorization")
		if authz == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
			return
		}
		tokenStr := parts[1]

		claims := jwt.MapClaims{}

		// Parse + validate signature + issuer + time claims (exp/nbf) with a leeway.
		parser := jwt.NewParser(
			jwt.WithIssuer(v.Issuer),
			jwt.WithLeeway(30*time.Second),
		)

		token, err := parser.ParseWithClaims(tokenStr, claims, v.Keyfunc)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		sub, _ := claims["sub"].(string)
		if strings.TrimSpace(sub) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token missing sub"})
			return
		}

		email, _ := claims["email"].(string)

		name := ""
		if vv, ok := claims["name"].(string); ok && vv != "" {
			name = vv
		} else if vv, ok := claims["preferred_username"].(string); ok && vv != "" {
			name = vv
		}

		roles := extractRolesFromClaims(claims)

		c.Set(CtxAuth, &models.AuthContext{
			Sub:   sub,
			Email: email,
			Name:  name,
			Roles: roles,
		})

		c.Next()
	}
}

func extractRolesFromClaims(claims jwt.MapClaims) []string {
	var roles []string

	// Keycloak usually: realm_access.roles
	if ra, ok := claims["realm_access"].(map[string]any); ok {
		if rr, ok := ra["roles"].([]any); ok {
			for _, r := range rr {
				if s, ok := r.(string); ok && s != "" {
					roles = append(roles, s)
				}
			}
		}
	}

	// sometimes: resource_access.<client>.roles
	if rsrc, ok := claims["resource_access"].(map[string]any); ok {
		for _, v := range rsrc {
			if m, ok := v.(map[string]any); ok {
				if rr, ok := m["roles"].([]any); ok {
					for _, r := range rr {
						if s, ok := r.(string); ok && s != "" {
							roles = append(roles, s)
						}
					}
				}
			}
		}
	}

	return dedupeStrings(roles)
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
