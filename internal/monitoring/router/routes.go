package router

import (
	"github.com/gin-gonic/gin"

	"github.com/picopubliccloud/alarm-api/internal/monitoring/handlers"
)

// RegisterRoutes registers monitoring/alarm/inventory/topology endpoints under a router group.
//
// Example usage from top-level router:
//   api := r.Group("/api")
//   monitoring := api.Group("") // keep same paths as before
//   router.RegisterRoutes(monitoring)
//
// Or (recommended):
//   monitoring := api.Group("/monitoring")
//   router.RegisterRoutes(monitoring)
func RegisterRoutes(api *gin.RouterGroup) {
	// ---- Alarm / Monitoring ----
	api.GET("/sites", handlers.GetSiteSummaries)
	api.GET("/down", handlers.GetDownDevices)
	api.GET("/psu", handlers.GetPSUHealth)
	api.GET("/asset-distribution", handlers.GetAssetDistribution)
	api.GET("/overview", handlers.GetOverview)
	api.GET("/devices", handlers.GetDevicesByIPs)

	// Links down + history + clearing
	api.GET("/down-links", handlers.GetDownLinks)
	api.GET("/device-history/:ip", handlers.GetDeviceHistory)
	api.GET("/link-history/:ip/:interface", handlers.GetLinkHistory)
	api.PATCH("/clear-link-status/:id", handlers.ClearLinkDownStatus)
	api.PATCH("/clear-device-alarm", handlers.ClearDeviceAlarm)
	api.GET("/all-device-alarm-history", handlers.GetAllDeviceAlarmHistory)
	api.GET("/all-link-alarm-history", handlers.GetAllLinkAlarmHistory)

	// ---- Excel exports ----
	api.GET("/down-device/download", handlers.DeviceDownExcel)
	api.GET("/down-link/download", handlers.LinkDownExcel)
	api.GET("/inventory/download", handlers.InventoryExcel)

	// ---- Inventory ----
	api.GET("/inventory_count", handlers.FetchInventoryCount)
	api.GET("/inventory", handlers.FetchInventory)
	api.POST("/add-inventory", handlers.AddInventory)
	api.PATCH("/update-inventory/:id", handlers.UpdateInventory)
	api.DELETE("/delete-inventory/:id", handlers.DeleteInventory)

	// ---- Topology ----
	api.GET("/get-nodes-edges/:district", handlers.GetNodesEdges)

	// ---- Zones ----
	api.GET("/get-zone-districts", handlers.GetZoneDistricts)

	// openstack
	api.GET("/openstack/projects", handlers.GetOpenStackProjects)
	api.GET("/openstack/public-ips", handlers.GetOpenStackPublicIPs)
	api.GET("/openstack/overview", handlers.GetOpenStackOverview)
}