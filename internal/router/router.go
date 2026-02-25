package router

import (
	"github.com/gin-gonic/gin"
	"database/sql"

	monitoringRouter "github.com/picopubliccloud/alarm-api/internal/monitoring/router"
	ticketingRouter "github.com/picopubliccloud/alarm-api/internal/ticketing/router"
)

func RegisterRoutes(r *gin.Engine, ticketDB *sql.DB) {
	api := r.Group("/api")

	// ---- Monitoring Module ----
	monitoring := api.Group("")
	monitoringRouter.RegisterRoutes(monitoring)

	// ---- Ticketing Module ----
	tickets := api.Group("/v1/tickets")
	ticketingRouter.RegisterRoutes(tickets, ticketDB)
}