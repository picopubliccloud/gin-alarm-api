package handlers

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	db "github.com/picopubliccloud/alarm-api/internal/data/actual_data"
	"github.com/picopubliccloud/alarm-api/internal/httpx"
	"github.com/picopubliccloud/alarm-api/internal/monitoring/models"
)

type Inventory = models.InventoryModel

func FetchInventoryCount(c *gin.Context) {
	ctx := c.Request.Context()

	var count int
	const query = "SELECT COUNT(*) FROM public.inventory WHERE status <> 'Deleted'"

	if err := db.DB.QueryRowContext(ctx, query).Scan(&count); err != nil {
		httpx.Internal(c, "failed to count inventory: "+err.Error())
		return
	}

	httpx.OK(c, gin.H{"total": count})
}

func FetchInventory(c *gin.Context) {
	ctx := c.Request.Context()

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 10 {
		limit = 10
	}

	offset := (page - 1) * limit

	var rowsOut []Inventory
	const query = `SELECT * FROM inventory WHERE status <> 'Deleted' ORDER BY id DESC LIMIT $1 OFFSET $2`

	rows, err := db.DB.QueryContext(ctx, query, limit, offset)
	if err != nil {
		httpx.Internal(c, "failed to fetch inventory data: "+err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item Inventory
		if err := rows.Scan(
			&item.ID, &item.AssetType, &item.AssetTag, &item.BrandModel, &item.NicLineCard, &item.PSU,
			&item.Site, &item.Rack, &item.Unit, &item.Owner, &item.Status, &item.Remark,
			&item.MgmtIPAddress, &item.SecondaryIP, &item.HostName, &item.OS, &item.OSVersion,
			&item.AddedDate, &item.AddedBy, &item.LastModifiedDate, &item.LastModifiedBy,
			&item.RemovedDate, &item.RemovedBy, &item.IsActive, &item.Purpose, &item.Verification,
			&item.UpDownStatus, &item.LastDownTime, &item.LastUpTime, &item.LastDownCheckTime,
			&item.TotalPowerSupplyCount, &item.UpPowerSupplyCount, &item.DownPowerSupplyCount, &item.LastPowerSupplyCheckTime,
		); err != nil {
			httpx.Internal(c, "failed to scan inventory row: "+err.Error())
			return
		}
		rowsOut = append(rowsOut, item)
	}

	if err := rows.Err(); err != nil {
		httpx.Internal(c, "row iteration error: "+err.Error())
		return
	}

	// Count (for pagination UI)
	var count int
	const countQuery = "SELECT COUNT(*) FROM public.inventory WHERE status <> 'Deleted'"
	if err := db.DB.QueryRowContext(ctx, countQuery).Scan(&count); err != nil {
		httpx.Internal(c, "failed to count inventory: "+err.Error())
		return
	}

	httpx.OK(c, gin.H{
		"count":   count,
		"results": rowsOut,
	})
}

func AddInventory(c *gin.Context) {
	ctx := c.Request.Context()

	var payload Inventory
	if err := c.ShouldBindJSON(&payload); err != nil {
		httpx.BadRequest(c, "invalid json body: "+err.Error())
		return
	}

	// Validate IPs (only if present)
	if payload.MgmtIPAddress != nil && *payload.MgmtIPAddress != "" && net.ParseIP(*payload.MgmtIPAddress) == nil {
		httpx.BadRequest(c, "invalid mgmt_ip_address")
		return
	}
	if payload.SecondaryIP != nil && *payload.SecondaryIP != "" && net.ParseIP(*payload.SecondaryIP) == nil {
		httpx.BadRequest(c, "invalid secondary_ip")
		return
	}

	currentTime := time.Now().UTC()

	const insertQuery = `
		INSERT INTO inventory (
			id,
			asset_type,
			asset_tag,
			brand_model,
			nic_line_card,
			psu,
			site,
			rack,
			unit,
			owner,
			status,
			remark,
			mgmt_ip_address,
			secondary_ip,
			host_name,
			os,
			os_version,
			added_date,
			added_by,
			last_modified_date,
			last_modified_by,
			removed_date,
			removed_by,
			is_active,
			purpose,
			verification,
			up_down_status,
			last_down_time,
			last_up_time,
			last_down_check_time,
			total_power_supply_count,
			up_power_supply_count,
			down_power_supply_count,
			last_power_supply_check_time
		) VALUES (
			$1,  $2,  $3,  $4,  $5,  $6,  $7,  $8,  $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34
		)
	`

	_, err := db.DB.ExecContext(ctx, insertQuery,
		payload.ID,
		payload.AssetType,
		payload.AssetTag,
		payload.BrandModel,
		payload.NicLineCard,
		payload.PSU,
		payload.Site,
		payload.Rack,
		payload.Unit,
		payload.Owner,
		payload.Status,
		payload.Remark,
		payload.MgmtIPAddress,
		payload.SecondaryIP,
		payload.HostName,
		payload.OS,
		payload.OSVersion,
		currentTime, // added_date
		payload.AddedBy,
		payload.LastModifiedDate,
		payload.LastModifiedBy,
		payload.RemovedDate,
		payload.RemovedBy,
		payload.IsActive,
		payload.Purpose,
		payload.Verification,
		payload.UpDownStatus,
		payload.LastDownTime,
		payload.LastUpTime,
		payload.LastDownCheckTime,
		payload.TotalPowerSupplyCount,
		payload.UpPowerSupplyCount,
		payload.DownPowerSupplyCount,
		payload.LastPowerSupplyCheckTime,
	)
	if err != nil {
		httpx.Internal(c, "failed to insert inventory record: "+err.Error())
		return
	}

	httpx.OK(c, gin.H{
		"message": "Data Recorded Successfully!",
		"data":    payload,
	})
}

func UpdateInventory(c *gin.Context) {
	ctx := c.Request.Context()

	paramID := c.Param("id")
	id, err := strconv.Atoi(paramID)
	if err != nil {
		httpx.BadRequest(c, "invalid id: "+err.Error())
		return
	}

	var payload Inventory
	if err := c.ShouldBindJSON(&payload); err != nil {
		httpx.BadRequest(c, "invalid json body: "+err.Error())
		return
	}

	if payload.MgmtIPAddress != nil && *payload.MgmtIPAddress != "" && net.ParseIP(*payload.MgmtIPAddress) == nil {
		httpx.BadRequest(c, "invalid mgmt_ip_address")
		return
	}
	if payload.SecondaryIP != nil && *payload.SecondaryIP != "" && net.ParseIP(*payload.SecondaryIP) == nil {
		httpx.BadRequest(c, "invalid secondary_ip")
		return
	}

	currentTime := time.Now().UTC()

	const updateQuery = `
		UPDATE inventory SET
			asset_type = $1,
			asset_tag = $2,
			brand_model = $3,
			nic_line_card = $4,
			psu = $5,
			site = $6,
			rack = $7,
			unit = $8,
			owner = $9,
			remark = $10,
			mgmt_ip_address = $11,
			secondary_ip = $12,
			host_name = $13,
			os = $14,
			os_version = $15,
			is_active = $16,
			purpose = $17,
			verification = $18,
			last_modified_by = $19,
			last_modified_date = $20
		WHERE id = $21;
	`

	res, err := db.DB.ExecContext(ctx, updateQuery,
		payload.AssetType,
		payload.AssetTag,
		payload.BrandModel,
		payload.NicLineCard,
		payload.PSU,
		payload.Site,
		payload.Rack,
		payload.Unit,
		payload.Owner,
		payload.Remark,
		payload.MgmtIPAddress,
		payload.SecondaryIP,
		payload.HostName,
		payload.OS,
		payload.OSVersion,
		payload.IsActive,
		payload.Purpose,
		payload.Verification,
		payload.LastModifiedBy,
		currentTime,
		id,
	)
	if err != nil {
		httpx.Internal(c, "failed to update record: "+err.Error())
		return
	}

	if n, _ := res.RowsAffected(); n == 0 {
		httpx.NotFound(c, fmt.Sprintf("record not found: id=%d", id))
		return
	}

	httpx.OK(c, gin.H{
		"message": "Data Updated Successfully!",
		"data":    payload,
	})
}

func DeleteInventory(c *gin.Context) {
	ctx := c.Request.Context()

	paramID := c.Param("id")
	id, err := strconv.Atoi(paramID)
	if err != nil {
		httpx.BadRequest(c, "invalid id: "+err.Error())
		return
	}

	removedBy := c.Query("removed_by")

	var status string
	const findQuery = "SELECT status FROM inventory WHERE id = $1 LIMIT 1"
	err = db.DB.QueryRowContext(ctx, findQuery, id).Scan(&status)

	if err == sql.ErrNoRows {
		httpx.NotFound(c, fmt.Sprintf("record not found: id=%d", id))
		return
	}
	if err != nil {
		httpx.Internal(c, "failed to find record: "+err.Error())
		return
	}

	if status == "Deleted" {
		httpx.OK(c, gin.H{"message": "Record Already Deleted!"})
		return
	}

	currentTime := time.Now().UTC()

	const delQuery = `
		UPDATE inventory
		SET status = 'Deleted', removed_by = $1, removed_date = $2
		WHERE id = $3
	`
	res, err := db.DB.ExecContext(ctx, delQuery, removedBy, currentTime, id)
	if err != nil {
		httpx.Internal(c, "failed to update status: "+err.Error())
		return
	}

	if n, _ := res.RowsAffected(); n == 0 {
		httpx.NotFound(c, fmt.Sprintf("record not found: id=%d", id))
		return
	}

	httpx.OK(c, gin.H{"message": "Status Updated!"})
}