// /internal/ticketing/handlers/meta.go
package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/repo"
)

type MetaHandler struct {
	Repo *repo.MetaRepo
}

func NewMetaHandler(r *repo.MetaRepo) *MetaHandler {
	return &MetaHandler{Repo: r}
}

func parseBool(s string, defaultVal bool) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return defaultVal
	}
	return s == "1" || s == "true" || s == "yes" || s == "y"
}

// GET /api/v1/tickets/meta/customers?active=true
func (h *MetaHandler) ListCustomers(c *gin.Context) {
	onlyActive := parseBool(c.Query("active"), false)

	items, err := h.Repo.ListCustomers(c.Request.Context(), onlyActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list customers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GET /api/v1/tickets/meta/services?active=true
func (h *MetaHandler) ListServices(c *gin.Context) {
	onlyActive := parseBool(c.Query("active"), true)

	items, err := h.Repo.ListServices(c.Request.Context(), onlyActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list services"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GET /api/v1/tickets/meta/users?q=tasnuva&active=true&only_ops=true&limit=80
func (h *MetaHandler) ListUsers(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	active := c.DefaultQuery("active", "true") == "true"
	onlyOps := c.DefaultQuery("only_ops", "true") == "true"

	limit := 80
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	items, err := h.Repo.ListUsers(c.Request.Context(), q, active, onlyOps, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}