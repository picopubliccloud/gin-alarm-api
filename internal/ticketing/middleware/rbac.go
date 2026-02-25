package middleware

import (
	"net/http"
	// "fmt"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/service"
)

// RBAC middleware (JWT-based):
// - reads external subject ONLY from gin context (set by UserUpsert/AuthContext)
// - resolves RBAC from DB
// - stores results in gin context
func RBAC(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Must be set by UserUpsert (which is set from token claims)
		sub := c.GetString(CtxExternalSubKey)
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing subject in context"})
			return
		}

		rbac, err := authSvc.ResolveRBAC(c.Request.Context(), sub)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		// Keep values typed (safer for handlers/services)
		c.Set(CtxExternalSubKey, rbac.ExternalSubject)
		c.Set(CtxUserIDKey, rbac.UserID)
		c.Set(CtxActorKey, rbac.Actor) // models.ActorType
		c.Set(CtxAllowedProjectsKey, rbac.AllowedProjectIDs)
		c.Set(CtxMaxVisibilityKey, rbac.MaxVisibility) // models.TicketVisibility
		c.Set(CtxCapabilitiesKey, rbac.Capabilities)   // repo.UserCapabilities

		c.Next()
	}
}

// Helpers for handlers (optional but makes handler code clean)

func GetActor(c *gin.Context) models.ActorType {
	if v, ok := c.Get(CtxActorKey); ok {
		// fmt.Printf("GetActor: ",v)
		if a, ok := v.(models.ActorType); ok {
			return a
		}
		if s, ok := v.(string); ok && s != "" {
			return models.ActorType(s)
		}
	}
	return models.ActorType("CUSTOMER")
}

func GetMaxVisibility(c *gin.Context) models.TicketVisibility {
	if v, ok := c.Get(CtxMaxVisibilityKey); ok {
		if mv, ok := v.(models.TicketVisibility); ok {
			return mv
		}
		if s, ok := v.(string); ok && s != "" {
			return models.TicketVisibility(s)
		}
	}
	return models.VisibilityPublic
}
func IsOpsActor(a models.ActorType) bool {
	s := strings.ToUpper(strings.TrimSpace(string(a)))
	switch s {
	case
		"CAREOPS", "CAREOPS_ADMIN", "CAREOPS_MANAGER",
		"PS", "PS_ADMIN", "PS_MANAGER",
		"SECOPS", "SECOPS_ADMIN", "SECOPS_MANAGER",
		"NETOPS", "NETOPS_ADMIN", "NETOPS_MANAGER":
		return true
	default:
		return false
	}
}

func GetAllowedProjectIDs(c *gin.Context) []string {
	return c.GetStringSlice(CtxAllowedProjectsKey)
}