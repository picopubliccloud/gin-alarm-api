package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/middleware"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
)

// Me returns the authenticated identity as seen by the backend after JWT validation + UserUpsert + RBAC.
func (h *TicketsHandler) Me(c *gin.Context) {
	v, ok := c.Get(middleware.CtxAuth)
	if !ok || v == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth context"})
		return
	}

	ac, ok := v.(*models.AuthContext)
	if !ok || ac == nil || ac.Sub == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth context"})
		return
	}

	userID := c.GetString(middleware.CtxUserIDKey)
	actor := middleware.GetActor(c)

	allowed, _ := c.Get(middleware.CtxAllowedProjectsKey)
	maxVis := middleware.GetMaxVisibility(c)

	c.JSON(http.StatusOK, gin.H{
		"user_id":          userID,
		"external_subject": ac.Sub,
		"email":            ac.Email,
		"display_name":     ac.Name,
		"roles":            ac.Roles,
		"actor_type":       actor,
		"allowed_projects": allowed,
		"max_visibility":   maxVis,
	})
}
