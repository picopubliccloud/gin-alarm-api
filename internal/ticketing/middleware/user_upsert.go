// /data/gin-alarm-api/internal/ticketing/middleware/user_upsert.go
package middleware

import (
	// "fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/service"
)

// Context key for Keycloak roles/groups as-is (no normalization).
// Use this in handlers for group-based RBAC.
const CtxUserGroupsKey = "ticketing.user_groups"

// pickActorFromGroups maps Keycloak groups/roles to your DB actor_type enum.
// - Keeps input "as-is" (case-sensitive match).
// - Picks the highest privilege if multiple groups are present.
func pickActorFromGroups(groups []string) models.ActorType {
	// Allowed values (must exist in ops.actor_type enum)
	valid := map[string]models.ActorType{
		// CAREOPS
		"CAREOPS":         models.ActorType("CAREOPS"),
		"CAREOPS_ADMIN":   models.ActorType("CAREOPS_ADMIN"),
		"CAREOPS_MANAGER": models.ActorType("CAREOPS_MANAGER"),

		// PS
		"PS":         models.ActorType("PS"),
		"PS_ADMIN":   models.ActorType("PS_ADMIN"),
		"PS_MANAGER": models.ActorType("PS_MANAGER"),

		// SECOPS
		"SECOPS":         models.ActorType("SECOPS"),
		"SECOPS_ADMIN":   models.ActorType("SECOPS_ADMIN"),
		"SECOPS_MANAGER": models.ActorType("SECOPS_MANAGER"),

		// NETOPS
		"NETOPS":         models.ActorType("NETOPS"),
		"NETOPS_ADMIN":   models.ActorType("NETOPS_ADMIN"),
		"NETOPS_MANAGER": models.ActorType("NETOPS_MANAGER"),
	}

	// Priority: prefer MANAGER > ADMIN > base role, and CAREOPS/PS/SECOPS/NETOPS order as you like.
	priority := []string{
		"CAREOPS_MANAGER", "CAREOPS_ADMIN", "CAREOPS",
		"PS_MANAGER", "PS_ADMIN", "PS",
		"SECOPS_MANAGER", "SECOPS_ADMIN", "SECOPS",
		"NETOPS_MANAGER", "NETOPS_ADMIN", "NETOPS",
	}

	seen := make(map[string]bool, len(groups))
	for _, g := range groups {
		seen[g] = true
	}

	for _, p := range priority {
		if seen[p] {
			if a, ok := valid[p]; ok {
				return a
			}
		}
	}

	// Fallback if no group matched
	return models.ActorType("CUSTOMER")
}

// UserUpsert middleware (JWT-based):
// - reads identity from AuthContext (CtxAuth), NOT from client headers
// - upserts ops.users by external_subject (Keycloak sub)
// - stores user_id, external_subject, user_groups into gin context
// NOTE: actor_type is derived from Keycloak groups/roles (ac.Roles). Defaults to CUSTOMER.
func UserUpsert(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// AuthContext middleware must run before this and set CtxAuth
		v, ok := c.Get(CtxAuth)
		if !ok || v == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth context"})
			return
		}

		ac, ok := v.(*models.AuthContext)
		if !ok || ac == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth context"})
			return
		}

		sub := strings.TrimSpace(ac.Sub)
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token missing sub"})
			return
		}

		// Keep roles/groups exactly as they come from Keycloak (no normalization)
		groups := ac.Roles

		// Optional email/display name from token claims
		var emailPtr *string
		if vv := strings.TrimSpace(ac.Email); vv != "" {
			emailPtr = &vv
		}

		var namePtr *string
		if vv := strings.TrimSpace(ac.Name); vv != "" {
			namePtr = &vv
		}

		actor := pickActorFromGroups(groups)

		// DEBUG (optional)
		// fmt.Printf("==> user_upsert sub=%s groups=%v actor=%s\n", sub, groups, string(actor))

		userID, err := authSvc.UpsertUser(c.Request.Context(), sub, emailPtr, namePtr, actor)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Store for later middleware/handlers
		c.Set(CtxExternalSubKey, sub)
		c.Set(CtxUserIDKey, userID)
		c.Set(CtxUserGroupsKey, groups)

		c.Next()
	}
}