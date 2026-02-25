package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	db "github.com/picopubliccloud/alarm-api/internal/data/actual_data"
	"github.com/picopubliccloud/alarm-api/internal/httpx"
	"github.com/picopubliccloud/alarm-api/internal/monitoring/models"
	"github.com/picopubliccloud/alarm-api/internal/utils"
)

// Device Down Table:
func DeviceDownExcel(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT alarm_id, hostname, ipaddress, alarm_status, severity, create_date,
		       site, comments, clear_by, acknowledge_by, clear_date, acknowledge_date
		FROM public.device_down
	`)
	if err != nil {
		httpx.Internal(c, "failed to fetch down devices: "+err.Error())
		return
	}
	defer rows.Close()

	type DownDevice struct {
		AlarmID         *string    `json:"alarm_id"`
		Name            *string    `json:"name"` // Hostname
		IP              *string    `json:"ip"`
		AlarmStatus     *string    `json:"alarm_status"`
		Severity        *string    `json:"severity"`
		DownSince       *time.Time `json:"down_since"`
		Site            *string    `json:"site"`
		Comments        *string    `json:"comments"`
		ClearedBy       *string    `json:"clear_by"`
		AcknowledgeBy   *string    `json:"acknowledge_by"`
		ClearDate       *time.Time `json:"clear_date"`
		AcknowledgeDate *time.Time `json:"acknowledge_date"`
	}

	var devices []DownDevice
	for rows.Next() {
		var row DownDevice

		if err := rows.Scan(
			&row.AlarmID, &row.Name, &row.IP, &row.AlarmStatus, &row.Severity, &row.DownSince,
			&row.Site, &row.Comments, &row.ClearedBy, &row.AcknowledgeBy, &row.ClearDate, &row.AcknowledgeDate,
		); err != nil {
			httpx.Internal(c, "failed to scan rows: "+err.Error())
			return
		}

		devices = append(devices, row)
	}

	if err := rows.Err(); err != nil {
		httpx.Internal(c, "row iteration error: "+err.Error())
		return
	}

	buffer, err := utils.GenerateExcel(devices)
	if err != nil {
		httpx.Internal(c, "failed to generate excel file: "+err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=down_devices.xlsx")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	c.Data(
		http.StatusOK,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		buffer.Bytes(),
	)
}

// Link down excel download:
func LinkDownExcel(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT alarm_id, alarm_name, hostname, ipaddress, interface,
		       link_type, description, link_status, alarm_status, severity,
		       create_date, clear_date, acknowledge_date, last_sync_date, comments,
		       acknowledge_by, clear_by, duration, flap_count, site
		FROM public.link_down
	`)
	if err != nil {
		httpx.Internal(c, "failed to fetch down links: "+err.Error())
		return
	}
	defer rows.Close()

	type LinkDown struct {
		ID              int        `json:"id"`
		AlarmID         *string    `json:"alarm_id"`
		AlarmName       *string    `json:"alarm_name"`
		Hostname        *string    `json:"hostname"`
		IPAddress       *string    `json:"ipaddress"`
		Interface       *string    `json:"interface"`
		LinkType        *string    `json:"link_type"`
		Description     *string    `json:"description"`
		LinkStatus      *string    `json:"link_status"`
		AlarmStatus     *string    `json:"alarm_status"`
		Severity        *string    `json:"severity"`
		CreateDate      *time.Time `json:"create_date"`
		ClearDate       *time.Time `json:"clear_date"`
		AcknowledgeDate *time.Time `json:"acknowledge_date"`
		LastSyncDate    *time.Time `json:"last_sync_date"`
		Comments        *string    `json:"comments"`
		AcknowledgeBy   *string    `json:"acknowledge_by"`
		ClearBy         *string    `json:"clear_by"`
		Duration        *string    `json:"duration"`
		FlapCount       *int       `json:"flap_count"`
		Site            *string    `json:"site"`
	}

	var links []LinkDown
	for rows.Next() {
		var row LinkDown

		if err := rows.Scan(
			&row.AlarmID, &row.AlarmName, &row.Hostname, &row.IPAddress, &row.Interface, &row.LinkType,
			&row.Description, &row.LinkStatus, &row.AlarmStatus, &row.Severity, &row.CreateDate,
			&row.ClearDate, &row.AcknowledgeDate, &row.LastSyncDate, &row.Comments,
			&row.AcknowledgeBy, &row.ClearBy, &row.Duration, &row.FlapCount, &row.Site,
		); err != nil {
			httpx.Internal(c, "failed to scan rows: "+err.Error())
			return
		}

		links = append(links, row)
	}

	if err := rows.Err(); err != nil {
		httpx.Internal(c, "row iteration error: "+err.Error())
		return
	}

	buffer, err := utils.GenerateExcel(links)
	if err != nil {
		httpx.Internal(c, "failed to generate excel file: "+err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=down_links.xlsx")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	c.Data(
		http.StatusOK,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		buffer.Bytes(),
	)
}

func InventoryExcel(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT asset_tag, asset_type, brand_model, nic_line_card, psu, site, rack, unit, owner, status,
		       remark, mgmt_ip_address, secondary_ip, host_name, os, os_version, added_date, added_by, last_modified_date,
		       last_modified_by, removed_date, removed_by, is_active, purpose, verification, up_down_status, last_down_time,
		       last_up_time, last_down_check_time, total_power_supply_count, up_power_supply_count, down_power_supply_count, last_power_supply_check_time
		FROM public.inventory
	`)
	if err != nil {
		httpx.Internal(c, "failed to fetch inventory data: "+err.Error())
		return
	}
	defer rows.Close()

	var inventory []models.InventoryModel
	for rows.Next() {
		var row models.InventoryModel

		if err := rows.Scan(
			&row.AssetTag, &row.AssetType, &row.BrandModel, &row.NicLineCard, &row.PSU, &row.Site, &row.Rack, &row.Unit,
			&row.Owner, &row.Status, &row.Remark, &row.MgmtIPAddress, &row.SecondaryIP, &row.HostName, &row.OS, &row.OSVersion,
			&row.AddedDate, &row.AddedBy, &row.LastModifiedDate, &row.LastModifiedBy, &row.RemovedDate, &row.RemovedBy,
			&row.IsActive, &row.Purpose, &row.Verification, &row.UpDownStatus, &row.LastDownTime, &row.LastUpTime,
			&row.LastDownCheckTime, &row.TotalPowerSupplyCount, &row.UpPowerSupplyCount, &row.DownPowerSupplyCount, &row.LastPowerSupplyCheckTime,
		); err != nil {
			httpx.Internal(c, "failed to scan rows: "+err.Error())
			return
		}

		inventory = append(inventory, row)
	}

	if err := rows.Err(); err != nil {
		httpx.Internal(c, "row iteration error: "+err.Error())
		return
	}

	buffer, err := utils.GenerateExcel(inventory)
	if err != nil {
		httpx.Internal(c, "failed to generate excel file: "+err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=inventory.xlsx")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	c.Data(
		http.StatusOK,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		buffer.Bytes(),
	)
}