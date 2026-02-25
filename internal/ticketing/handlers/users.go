package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	tMW "github.com/picopubliccloud/alarm-api/internal/ticketing/middleware"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/service"
)

type UsersHandler struct {
	Auth *service.AuthService
}

func NewUsersHandler(auth *service.AuthService) *UsersHandler {
	return &UsersHandler{Auth: auth}
}

// GET /api/ticketing/me
func (h *UsersHandler) Me(c *gin.Context) {
	userID, _ := c.Get(tMW.CtxUserIDKey)
	actor, _ := c.Get(tMW.CtxActorKey)
	sub, _ := c.Get(tMW.CtxExternalSubKey)

	rbacAny, _ := c.Get(tMW.CtxCapabilitiesKey)
	rbac, _ := rbacAny.(*service.RBACContext)

	resp := gin.H{
		"user_id":           userID,
		"actor_type":        actor,
		"external_subject":  sub,
		"allowed_projects":  c.GetStringSlice(tMW.CtxAllowedProjectsKey),
		"max_visibility":    c.GetString(tMW.CtxMaxVisibilityKey),
	}

	if rbac != nil {
		caps := rbac.Capabilities
		resp["capabilities"] = gin.H{
			"can_view_restricted":  caps.CanViewRestricted,
			"can_manage_sla":       caps.CanManageSLA,
			"can_merge_tickets":    caps.CanMergeTickets,
			"can_assign_any":       caps.CanAssignAny,
			"can_declare_incident": caps.CanDeclareIncident,
		}
	}

	c.JSON(http.StatusOK, resp)
}
