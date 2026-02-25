package handlers

import (
	"net/http"
	"strconv"
	"strings"


	"github.com/gin-gonic/gin"
	tMW "github.com/picopubliccloud/alarm-api/internal/ticketing/middleware"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/repo"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/service"
)

type TicketsHandler struct {
	Tickets *service.TicketService
}

func NewTicketsHandler(tickets *service.TicketService) *TicketsHandler {
	return &TicketsHandler{Tickets: tickets}
}

func (h *TicketsHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GET /api/v1/tickets?limit=50&cursor=12345&status=NEW&severity=HIGH
func (h *TicketsHandler) ListTickets(c *gin.Context) {
	allowedProjects := c.GetStringSlice(tMW.CtxAllowedProjectsKey)
	maxVis := tMW.GetMaxVisibility(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	var cursorNum int64
	if cur := c.Query("cursor"); cur != "" {
		cursorNum, _ = strconv.ParseInt(cur, 10, 64)
	}

	filter := repo.TicketListFilter{
		AllowedProjectIDs:  allowedProjects,
		MaxVisibility:      maxVis,
		Limit:              limit,
		CursorTicketNumber: cursorNum,
	}

	if st := c.Query("status"); st != "" {
		filter.Status = models.TicketStatus(st)
	}
	if sv := c.Query("severity"); sv != "" {
		filter.Severity = models.TicketSeverity(sv)
	}

	items, nextCursor, hasMore, err := h.Tickets.ListTickets(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

// GET /api/v1/tickets/pool?limit=50&cursor=12345&status=NEW&severity=HIGH
func (h *TicketsHandler) ListPoolTickets(c *gin.Context) {
	allowedProjects := c.GetStringSlice(tMW.CtxAllowedProjectsKey)
	maxVis := tMW.GetMaxVisibility(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	var cursorNum int64
	if cur := c.Query("cursor"); cur != "" {
		cursorNum, _ = strconv.ParseInt(cur, 10, 64)
	}

	filter := repo.TicketListFilter{
		AllowedProjectIDs:  allowedProjects,
		MaxVisibility:      maxVis,
		Limit:              limit,
		CursorTicketNumber: cursorNum,
		OnlyPool:           true, // pool = no active OWNER assignment
	}

	if st := c.Query("status"); st != "" {
		filter.Status = models.TicketStatus(st)
	}
	if sv := c.Query("severity"); sv != "" {
		filter.Severity = models.TicketSeverity(sv)
	}

	items, nextCursor, hasMore, err := h.Tickets.ListPoolTickets(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

// GET /api/v1/tickets/:ticket_id
func (h *TicketsHandler) GetTicket(c *gin.Context) {
	ticketID := c.Param("ticket_id")

	allowedProjects := c.GetStringSlice(tMW.CtxAllowedProjectsKey)
	maxVis := tMW.GetMaxVisibility(c)

	d, err := h.Tickets.GetTicket(c.Request.Context(), ticketID, allowedProjects, maxVis)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, d)
}

// POST /api/v1/tickets
func (h *TicketsHandler) CreateTicket(c *gin.Context) {
	var req models.CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString(tMW.CtxUserIDKey)
	actor := tMW.GetActor(c)   

	ticketID, ticketNumber, err := h.Tickets.CreateTicket(c.Request.Context(), &req, userID, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"ticket_id":     ticketID,
		"ticket_number": ticketNumber,
	})
}

// POST /api/v1/tickets/:ticket_id/updates
func (h *TicketsHandler) AddUpdate(c *gin.Context) {
	ticketID := c.Param("ticket_id")

	var req models.AddUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString(tMW.CtxUserIDKey)
	actor := tMW.GetActor(c)   

	if err := h.Tickets.AddUpdate(c.Request.Context(), ticketID, &req, userID, actor); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

type assignOwnerRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// POST /api/v1/tickets/:ticket_id/assign
func (h *TicketsHandler) AssignOwner(c *gin.Context) {
	ticketID := c.Param("ticket_id")

	var req assignOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	actorUserID := c.GetString(tMW.CtxUserIDKey)

	if err := h.Tickets.AssignOwner(c.Request.Context(), ticketID, req.UserID, actorUserID); err != nil {
		// sql.ErrNoRows may indicate locked
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

type unassignOwnerRequest struct {
	UnassignReasonID *int16  `json:"unassign_reason_id"`
	Note             *string `json:"note"`
}


// POST /api/v1/tickets/:ticket_id/unassign
func (h *TicketsHandler) UnassignOwner(c *gin.Context) {
	ticketID := c.Param("ticket_id")
	actorUserID := c.GetString(tMW.CtxUserIDKey)

	var req unassignOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// allow empty body (no reason/note)
		req = unassignOwnerRequest{}
	}

	if err := h.Tickets.UnassignOwner(c.Request.Context(), ticketID, req.UnassignReasonID, req.Note, actorUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

type lockRequest struct {
	LockReasonID int16 `json:"lock_reason_id"`
	TTLSeconds   *int64 `json:"ttl_seconds"`
}

// POST /api/v1/tickets/:ticket_id/lock
func (h *TicketsHandler) LockTicket(c *gin.Context) {
	ticketID := c.Param("ticket_id")
	userID := c.GetString(tMW.CtxUserIDKey)

	var req lockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = lockRequest{}
	}

	ttl := int64(30 * 60)
	if req.TTLSeconds != nil && *req.TTLSeconds > 0 {
		ttl = *req.TTLSeconds
	}

	if err := h.Tickets.LockTicket(c.Request.Context(), ticketID, userID, req.LockReasonID, ttl ); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// POST /api/v1/tickets/:ticket_id/unlock
func (h *TicketsHandler) UnlockTicket(c *gin.Context) {
	ticketID := c.Param("ticket_id")
	userID := c.GetString(tMW.CtxUserIDKey)

	if err := h.Tickets.UnlockTicket(c.Request.Context(), ticketID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// POST /api/v1/tickets/:ticket_id/close
func (h *TicketsHandler) CloseTicket(c *gin.Context) {
	ticketID := c.Param("ticket_id")
	userID := c.GetString(tMW.CtxUserIDKey)

	var req models.CloseTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Tickets.CloseTicket(c.Request.Context(), ticketID, &req, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *TicketsHandler) SummaryTickets(c *gin.Context) {
	pool := strings.EqualFold(c.Query("pool"), "true")
	status := strings.TrimSpace(c.Query("status"))
	severity := strings.TrimSpace(c.Query("severity"))
	includeClosed := strings.EqualFold(c.Query("include_closed"), "true")

	actor := tMW.GetActor(c)

	allowedProjects := c.GetStringSlice(tMW.CtxAllowedProjectsKey)
	maxVis := tMW.GetMaxVisibility(c)

	// ✅ OPS can see ALL tickets and ALL summaries
	if tMW.IsOpsActor(actor) {
		allowedProjects = nil                      // no project filter
		maxVis = models.TicketVisibility("RESTRICTED") // your repo treats else-case as "all"
		// If you have a constant, use it:
		// maxVis = models.VisibilityRestricted
	}

	resp, err := h.Tickets.SummaryTickets(
		c.Request.Context(),
		pool,
		status,
		severity,
		includeClosed,
		allowedProjects,
		maxVis,
	)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
