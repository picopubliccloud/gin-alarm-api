package handlers

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	db "github.com/picopubliccloud/alarm-api/internal/data/actual_data"
	data "github.com/picopubliccloud/alarm-api/internal/data/mock_data"
	"github.com/picopubliccloud/alarm-api/internal/httpx"
	"github.com/picopubliccloud/alarm-api/internal/monitoring/models"
)

func GetSiteSummaries(c *gin.Context) {
	type SiteSummaryWithDown struct {
		Site           string   `json:"site"`
		Total          int      `json:"total"`
		TotalActive    int      `json:"totalactive"`
		Up             int      `json:"up"`
		Down           int      `json:"down"`
		TotalIPs       []string `json:"total_ips"`
		TotalActiveIPs []string `json:"total_active_ips"`
		UpIPs          []string `json:"up_ips"`
		DownIPs        []string `json:"down_ips"`
	}

	ctx := c.Request.Context()

	// 1) totals per site
	rows, err := db.DB.QueryContext(ctx, `
		SELECT site,
		       COUNT(*) AS total,
		       COUNT(CASE WHEN is_active = true THEN 1 END) AS active
		FROM public.inventory
		GROUP BY site
	`)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows.Close()

	sites := make(map[string]*SiteSummaryWithDown)
	for rows.Next() {
		var s SiteSummaryWithDown
		if err := rows.Scan(&s.Site, &s.Total, &s.TotalActive); err != nil {
			httpx.Internal(c, err.Error())
			return
		}
		s.TotalIPs = []string{}
		s.TotalActiveIPs = []string{}
		s.UpIPs = []string{}
		s.DownIPs = []string{}
		sites[s.Site] = &s
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, err.Error())
		return
	}

	// 2) all IPs from inventory
	rows2, err := db.DB.QueryContext(ctx, `
		SELECT site, mgmt_ip_address::text, is_active
		FROM public.inventory
	`)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var site, ip sql.NullString
		var isActive sql.NullBool
		if err := rows2.Scan(&site, &ip, &isActive); err != nil {
			httpx.Internal(c, err.Error())
			return
		}
		if !site.Valid || !ip.Valid {
			continue
		}
		ipStr := strings.Split(ip.String, "/")[0]

		s, ok := sites[site.String]
		if !ok {
			continue
		}
		s.TotalIPs = append(s.TotalIPs, ipStr)
		if isActive.Valid && isActive.Bool {
			s.TotalActiveIPs = append(s.TotalActiveIPs, ipStr)
		}
	}
	if err := rows2.Err(); err != nil {
		httpx.Internal(c, err.Error())
		return
	}

	// 3) down devices
	rows3, err := db.DB.QueryContext(ctx, `
		SELECT ipaddress, site
		FROM public.device_down
		WHERE alarm_status = 'ACTIVE'
	`)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows3.Close()

	for rows3.Next() {
		var ip, site sql.NullString
		if err := rows3.Scan(&ip, &site); err != nil {
			httpx.Internal(c, err.Error())
			return
		}
		if !ip.Valid || !site.Valid {
			continue
		}
		s, ok := sites[site.String]
		if !ok {
			continue
		}
		s.DownIPs = append(s.DownIPs, ip.String)
	}
	if err := rows3.Err(); err != nil {
		httpx.Internal(c, err.Error())
		return
	}

	// 4) calculate up/down
	for _, s := range sites {
		s.Down = len(s.DownIPs)
		s.Up = s.TotalActive - s.Down

		downSet := make(map[string]struct{}, len(s.DownIPs))
		for _, ip := range s.DownIPs {
			downSet[ip] = struct{}{}
		}
		for _, ip := range s.TotalActiveIPs {
			if _, found := downSet[ip]; !found {
				s.UpIPs = append(s.UpIPs, ip)
			}
		}
	}

	result := make([]SiteSummaryWithDown, 0, len(sites))
	for _, s := range sites {
		result = append(result, *s)
	}

	httpx.OK(c, gin.H{"sites": result})
}

// GET /down
func GetDownDevices(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT alarm_id, alarm_status, severity, site, hostname, ipaddress::text, create_date, comments,
		       clear_by, clear_date, acknowledge_by, acknowledge_date
		FROM public.device_down
		WHERE alarm_status = 'ACTIVE'
	`)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows.Close()

	var downDevices []models.DownDevice
	for rows.Next() {
		var d models.DownDevice
		var alarmID, alarmStatus, severity, hostName, ip, comments, clearBy, acknowledgeBy sql.NullString
		var downSince, clearDate, acknowledgeDate sql.NullTime

		if err := rows.Scan(
			&alarmID, &alarmStatus, &severity, &d.Site, &hostName, &ip, &downSince, &comments,
			&clearBy, &clearDate, &acknowledgeBy, &acknowledgeDate,
		); err != nil {
			httpx.Internal(c, err.Error())
			return
		}

		d.Name = "Unknown"
		if hostName.Valid {
			d.Name = hostName.String
		}
		if ip.Valid {
			d.IP = ip.String
		}
		if downSince.Valid {
			d.DownSince = downSince.Time
		}
		if comments.Valid {
			d.Comments = comments.String
		}
		if alarmID.Valid {
			d.AlarmID = alarmID.String
		}
		if alarmStatus.Valid {
			d.AlarmStatus = alarmStatus.String
		}
		if severity.Valid {
			d.Severity = severity.String
		}
		if clearBy.Valid {
			d.ClearedBy = clearBy.String
		}
		if clearDate.Valid {
			d.ClearDate = &clearDate.Time
		}
		if acknowledgeBy.Valid {
			d.AcknowledgeBy = acknowledgeBy.String
		}
		if acknowledgeDate.Valid {
			d.AcknowledgeDate = &acknowledgeDate.Time
		}

		downDevices = append(downDevices, d)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, err.Error())
		return
	}

	downIPs := make([]string, 0, len(downDevices))
	for _, d := range downDevices {
		if d.IP != "" {
			downIPs = append(downIPs, d.IP)
		}
	}

	httpx.OK(c, gin.H{
		"count":   len(downDevices),
		"devices": downDevices,
		"ips":     downIPs,
	})
}

// GET /api/psu
func GetPSUHealth(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT
			site,
			COUNT(*) AS total_devices,
			SUM(CASE WHEN CAST(total_power_supply_count AS integer) = 2 THEN 1 ELSE 0 END) AS redundant_psu_count,
			SUM(
				CASE
					WHEN CAST(total_power_supply_count AS integer) = 2
						AND CAST(down_power_supply_count AS integer) > 0
					THEN 1
					ELSE 0
				END
			) AS failed_psu_count
		FROM public.inventory
		WHERE is_active = true
		GROUP BY site
		ORDER BY site;
	`)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows.Close()

	type PSUEntry struct {
		Site      string   `json:"site"`
		Total     int      `json:"total"`
		Failed    int      `json:"failed"`
		Redundant int      `json:"redundant"`
		TotalIPs  []string `json:"total_ips"`
		FailedIPs []string `json:"failed_ips"`
		UpIPs     []string `json:"up_ips"`
	}

	psuHealth := []PSUEntry{}

	for rows.Next() {
		var site string
		var total, redundant, failed sql.NullInt64

		if err := rows.Scan(&site, &total, &redundant, &failed); err != nil {
			httpx.Internal(c, err.Error())
			return
		}

		entry := PSUEntry{
			Site:      site,
			TotalIPs:  []string{},
			FailedIPs: []string{},
			UpIPs:     []string{},
		}
		if total.Valid {
			entry.Total = int(total.Int64)
		}
		if failed.Valid {
			entry.Failed = int(failed.Int64)
		}
		if redundant.Valid {
			entry.Redundant = int(redundant.Int64)
		}

		// IMPORTANT: do NOT defer Close() in a loop
		rowsIPs, err := db.DB.QueryContext(ctx, `
			SELECT mgmt_ip_address::text, total_power_supply_count, down_power_supply_count
			FROM public.inventory
			WHERE site = $1 AND is_active = true
		`, site)
		if err != nil {
			httpx.Internal(c, err.Error())
			return
		}

		for rowsIPs.Next() {
			var ip sql.NullString
			var totalPSU, downPSU sql.NullInt64
			if err := rowsIPs.Scan(&ip, &totalPSU, &downPSU); err != nil {
				rowsIPs.Close()
				httpx.Internal(c, err.Error())
				return
			}
			if ip.Valid {
				entry.TotalIPs = append(entry.TotalIPs, ip.String)
				if downPSU.Valid && downPSU.Int64 > 0 {
					entry.FailedIPs = append(entry.FailedIPs, ip.String)
				} else {
					entry.UpIPs = append(entry.UpIPs, ip.String)
				}
			}
		}
		if err := rowsIPs.Err(); err != nil {
			rowsIPs.Close()
			httpx.Internal(c, err.Error())
			return
		}
		rowsIPs.Close()

		psuHealth = append(psuHealth, entry)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, err.Error())
		return
	}

	httpx.OK(c, gin.H{"psu_health": psuHealth})
}

// GET /asset-distribution
func GetAssetDistribution(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT asset_type, COUNT(*) AS value
		FROM public.inventory
		WHERE is_active = true
		GROUP BY asset_type
	`)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows.Close()

	type AssetEntry struct {
		Name  string   `json:"name"`
		Value int      `json:"value"`
		IPs   []string `json:"ips"`
	}

	assets := []AssetEntry{}

	for rows.Next() {
		var name string
		var value int
		if err := rows.Scan(&name, &value); err != nil {
			httpx.Internal(c, err.Error())
			return
		}

		ipRows, err := db.DB.QueryContext(ctx, `
			SELECT mgmt_ip_address::text
			FROM public.inventory
			WHERE asset_type = $1 AND is_active = true
		`, name)
		if err != nil {
			httpx.Internal(c, err.Error())
			return
		}

		ips := []string{}
		for ipRows.Next() {
			var ip sql.NullString
			if err := ipRows.Scan(&ip); err != nil {
				ipRows.Close()
				httpx.Internal(c, err.Error())
				return
			}
			if ip.Valid {
				ips = append(ips, ip.String)
			}
		}
		if err := ipRows.Err(); err != nil {
			ipRows.Close()
			httpx.Internal(c, err.Error())
			return
		}
		ipRows.Close()

		assets = append(assets, AssetEntry{Name: name, Value: value, IPs: ips})
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, err.Error())
		return
	}

	httpx.OK(c, gin.H{"asset_distribution": assets})
}

// GET /api/overview
func GetOverview(c *gin.Context) {
	var totalDevices, totalDown, totalUp int
	for _, site := range data.SiteSummaries {
		totalDevices += site.Total
		totalDown += site.Down
		totalUp += site.Up
	}
	httpx.OK(c, gin.H{
		"total_devices": totalDevices,
		"total_up":      totalUp,
		"total_down":    totalDown,
		"timestamp":     time.Now(),
	})
}

// GET /devices?ips=ip1,ip2,ip3
func GetDevicesByIPs(c *gin.Context) {
	ctx := c.Request.Context()

	ipsParam := c.Query("ips")
	if ipsParam == "" {
		httpx.BadRequest(c, "missing ips query parameter")
		return
	}

	ips := strings.Split(ipsParam, ",")

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 10 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := `
		SELECT id, asset_type, asset_tag, brand_model, nic_line_card, psu, site, rack, unit,
			   owner, status, remark, mgmt_ip_address, secondary_ip, host_name, os, os_version,
			   added_date, added_by, last_modified_date, last_modified_by, removed_date, removed_by,
			   is_active, purpose, verification, up_down_status, last_down_time, last_up_time,
			   last_down_check_time, total_power_supply_count, up_power_supply_count,
			   down_power_supply_count, last_power_supply_check_time
		FROM public.inventory
		WHERE mgmt_ip_address = ANY($1)
		LIMIT $2 OFFSET $3
	`

	rows, err := db.DB.QueryContext(ctx, query, pq.Array(ips), limit, offset)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		var removedDate, lastDownTime, lastUpTime, lastDownCheckTime, lastPSUCheckTime sql.NullTime

		if err := rows.Scan(
			&d.ID, &d.AssetType, &d.AssetTag, &d.BrandModel, &d.NICLineCard, &d.PSU, &d.Site, &d.Rack, &d.Unit,
			&d.Owner, &d.Status, &d.Remark, &d.MgmtIPAddress, &d.SecondaryIP, &d.HostName, &d.OS, &d.OSVersion,
			&d.AddedDate, &d.AddedBy, &d.LastModifiedDate, &d.LastModifiedBy, &removedDate, &d.RemovedBy,
			&d.IsActive, &d.Purpose, &d.Verification, &d.UpDownStatus, &lastDownTime, &lastUpTime,
			&lastDownCheckTime, &d.TotalPowerSupplyCount, &d.UpPowerSupplyCount, &d.DownPowerSupplyCount,
			&lastPSUCheckTime,
		); err != nil {
			httpx.Internal(c, err.Error())
			return
		}

		if removedDate.Valid {
			d.RemovedDate = &removedDate.Time
		}
		if lastDownTime.Valid {
			d.LastDownTime = &lastDownTime.Time
		}
		if lastUpTime.Valid {
			d.LastUpTime = &lastUpTime.Time
		}
		if lastDownCheckTime.Valid {
			d.LastDownCheckTime = &lastDownCheckTime.Time
		}
		if lastPSUCheckTime.Valid {
			d.LastPowerSupplyCheckTime = &lastPSUCheckTime.Time
		}

		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, err.Error())
		return
	}

	httpx.OK(c, gin.H{"devices": devices})
}

// GET /api/down-links
func GetDownLinks(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, alarm_id, alarm_name, hostname, ipaddress, interface, link_type, description,
			   link_status, alarm_status, severity, create_date, clear_date, acknowledge_date,
			   last_sync_date, comments, acknowledge_by, clear_by, duration, flap_count, site
		FROM public.link_down
		WHERE alarm_status = 'active'
		ORDER BY create_date DESC
	`)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows.Close()

	var results []models.LinkDown

	for rows.Next() {
		var l models.LinkDown
		var clearDate, ackDate, syncDate sql.NullTime
		var comments, ackBy, clearBy sql.NullString

		if err := rows.Scan(
			&l.ID, &l.AlarmID, &l.AlarmName, &l.Hostname, &l.IPAddress,
			&l.Interface, &l.LinkType, &l.Description, &l.LinkStatus,
			&l.AlarmStatus, &l.Severity, &l.CreateDate, &clearDate,
			&ackDate, &syncDate, &comments, &ackBy, &clearBy,
			&l.Duration, &l.FlapCount, &l.Site,
		); err != nil {
			httpx.Internal(c, err.Error())
			return
		}

		if clearDate.Valid {
			l.ClearDate = &clearDate.Time
		}
		if ackDate.Valid {
			l.AcknowledgeDate = &ackDate.Time
		}
		if syncDate.Valid {
			l.LastSyncDate = &syncDate.Time
		}
		if comments.Valid {
			l.Comments = comments.String
		}
		if ackBy.Valid {
			l.AcknowledgeBy = ackBy.String
		}
		if clearBy.Valid {
			l.ClearBy = clearBy.String
		}

		results = append(results, l)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, err.Error())
		return
	}

	httpx.OK(c, gin.H{
		"count": len(results),
		"links": results,
	})
}

// PATCH /api/clear-link-down/:id
func ClearLinkDownStatus(c *gin.Context) {
	ctx := c.Request.Context()

	paramID := c.Param("id")
	id, err := strconv.Atoi(paramID)
	if err != nil {
		httpx.BadRequest(c, "invalid id: "+err.Error())
		return
	}

	var payload models.LinkDown
	if err := c.ShouldBindJSON(&payload); err != nil {
		httpx.BadRequest(c, "invalid json body: "+err.Error())
		return
	}

	currentTime := time.Now().UTC()

	if payload.AlarmStatus == "active" {
		updateQuery := `
			UPDATE public.link_down SET
				link_status  = $1,
				alarm_status = $2,
				severity     = $3,
				comments     = $4
			WHERE id = $5;
		`
		_, err = db.DB.ExecContext(ctx, updateQuery,
			payload.LinkStatus,
			payload.AlarmStatus,
			payload.Severity,
			payload.Comments,
			id,
		)
	} else {
		updateQuery := `
			UPDATE public.link_down SET
				link_status  = $1,
				alarm_status = $2,
				severity     = $3,
				comments     = $4,
				clear_by     = $5,
				clear_date   = $6
			WHERE id = $7;
		`
		_, err = db.DB.ExecContext(ctx, updateQuery,
			payload.LinkStatus,
			payload.AlarmStatus,
			payload.Severity,
			payload.Comments,
			payload.ClearBy,
			currentTime,
			id,
		)
	}

	if err != nil {
		httpx.Internal(c, "failed to update link alarm status: "+err.Error())
		return
	}

	httpx.OK(c, gin.H{"message": "Link Alarm Status Cleared Successfully!"})
}

// PATCH /api/clear-device-alarm
func ClearDeviceAlarm(c *gin.Context) {
	ctx := c.Request.Context()

	var payload models.DownDevice
	if err := c.ShouldBindJSON(&payload); err != nil {
		httpx.BadRequest(c, "invalid json body: "+err.Error())
		return
	}

	currentTime := time.Now().UTC()

	var (
		res sql.Result
		err error
	)

	if payload.AlarmStatus == "ACTIVE" {
		updateQuery := `
			UPDATE public.device_down SET
				alarm_status = $1,
				severity     = $2,
				comments     = $3
			WHERE alarm_id = $4;
		`
		res, err = db.DB.ExecContext(ctx, updateQuery,
			payload.AlarmStatus,
			payload.Severity,
			payload.Comments,
			payload.AlarmID,
		)
	} else {
		updateQuery := `
			UPDATE public.device_down SET
				alarm_status = $1,
				severity     = $2,
				comments     = $3,
				clear_by     = $4,
				clear_date   = $5
			WHERE alarm_id = $6;
		`
		res, err = db.DB.ExecContext(ctx, updateQuery,
			payload.AlarmStatus,
			payload.Severity,
			payload.Comments,
			payload.ClearedBy,
			currentTime,
			payload.AlarmID,
		)
	}

	if err != nil {
		httpx.Internal(c, "failed to update device alarm status: "+err.Error())
		return
	}

	// Optional but useful: ensure something actually updated
	if n, _ := res.RowsAffected(); n == 0 {
		httpx.NotFound(c, fmt.Sprintf("alarm_id not found: %s", payload.AlarmID))
		return
	}

	httpx.OK(c, gin.H{"message": "Device Alarm Status Updated Successfully!"})
}

// TODO: History apis:
func GetDeviceHistory(c *gin.Context) {
	ctx := c.Request.Context()
	ipaddress := c.Param("ip")

	if net.ParseIP(ipaddress) == nil {
		httpx.BadRequest(c, "invalid ip address")
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 10 {
		limit = 10
	}

	offset := (page - 1) * limit

	rows, err := db.DB.QueryContext(ctx, `
		SELECT alarm_id, hostname, alarm_status, site, clear_date
		FROM public.device_down
		WHERE ipaddress = $1
		ORDER BY clear_date DESC
		LIMIT $2 OFFSET $3
	`, ipaddress, limit, offset)
	if err != nil {
		httpx.Internal(c, "database query failed: "+err.Error())
		return
	}
	defer rows.Close()

	type deviceData struct {
		AlarmID     *string    `json:"alarm_id"`
		Name        *string    `json:"name"`
		AlarmStatus *string    `json:"alarm_status"`
		Site        *string    `json:"site"`
		ClearDate   *time.Time `json:"clear_date"`
	}

	var result []deviceData
	for rows.Next() {
		var row deviceData
		if err := rows.Scan(&row.AlarmID, &row.Name, &row.AlarmStatus, &row.Site, &row.ClearDate); err != nil {
			httpx.Internal(c, "failed to scan row: "+err.Error())
			return
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, "row iteration error: "+err.Error())
		return
	}

	var count int
	if err := db.DB.QueryRowContext(ctx, `
		SELECT COUNT(id)
		FROM public.device_down
		WHERE ipaddress = $1
	`, ipaddress).Scan(&count); err != nil {
		httpx.Internal(c, "unable to fetch count: "+err.Error())
		return
	}

	httpx.OK(c, gin.H{
		"count":   count,
		"results": result,
	})
}

func GetLinkHistory(c *gin.Context) {
	ctx := c.Request.Context()
	ipaddress := c.Param("ip")
	inf := c.Param("interface")

	if net.ParseIP(ipaddress) == nil {
		httpx.BadRequest(c, "invalid ip address")
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 10 {
		limit = 10
	}

	offset := (page - 1) * limit

	rows, err := db.DB.QueryContext(ctx, `
		SELECT alarm_id, hostname, ipaddress, interface, alarm_status, clear_date, clear_by
		FROM public.link_down
		WHERE ipaddress = $1 AND interface = $2
		ORDER BY clear_date DESC
		LIMIT $3 OFFSET $4
	`, ipaddress, inf, limit, offset)
	if err != nil {
		httpx.Internal(c, "database query failed: "+err.Error())
		return
	}
	defer rows.Close()

	type linkData struct {
		AlarmID     *string    `json:"alarm_id"`
		Hostname    *string    `json:"hostname"`
		IPAddress   *string    `json:"ipaddress"`
		Interface   *string    `json:"interface"`
		AlarmStatus *string    `json:"alarm_status"`
		ClearDate   *time.Time `json:"clear_date"`
		ClearBy     *string    `json:"clear_by"`
	}

	var result []linkData
	for rows.Next() {
		var row linkData
		if err := rows.Scan(
			&row.AlarmID,
			&row.Hostname,
			&row.IPAddress,
			&row.Interface,
			&row.AlarmStatus,
			&row.ClearDate,
			&row.ClearBy,
		); err != nil {
			httpx.Internal(c, "failed to scan row: "+err.Error())
			return
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, "row iteration error: "+err.Error())
		return
	}

	var count int
	if err := db.DB.QueryRowContext(ctx, `
		SELECT COUNT(id)
		FROM public.link_down
		WHERE ipaddress = $1 AND interface = $2
	`, ipaddress, inf).Scan(&count); err != nil {
		httpx.Internal(c, "unable to fetch count: "+err.Error())
		return
	}

	httpx.OK(c, gin.H{
		"count":   count,
		"results": result,
	})
}

func GetAllDeviceAlarmHistory(c *gin.Context) {
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

	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	var startDate *time.Time
	var endDate *time.Time

	if startDateStr != "" {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			httpx.BadRequest(c, "invalid startDate format. Use YYYY-MM-DD")
			return
		}
		startDate = &t
	}

	if endDateStr != "" {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			httpx.BadRequest(c, "invalid endDate format. Use YYYY-MM-DD")
			return
		}
		t = t.Add(24*time.Hour - time.Nanosecond)
		endDate = &t
	}

	type deviceData struct {
		AlarmID     *string    `json:"alarm_id"`
		Name        *string    `json:"name"`
		IP          *string    `json:"ip"`
		AlarmStatus *string    `json:"alarm_status"`
		Site        *string    `json:"site"`
		ClearDate   *time.Time `json:"clear_date"`
	}

	rows, err := db.DB.QueryContext(ctx, `
		SELECT alarm_id, hostname, ipaddress, alarm_status, site, clear_date
		FROM public.device_down
		WHERE ($1::timestamp IS NULL OR clear_date >= $1)
		  AND ($2::timestamp IS NULL OR clear_date <= $2)
		ORDER BY clear_date
		LIMIT $3 OFFSET $4
	`, startDate, endDate, limit, offset)
	if err != nil {
		httpx.Internal(c, "database query failed: "+err.Error())
		return
	}
	defer rows.Close()

	var result []deviceData
	for rows.Next() {
		var row deviceData
		if err := rows.Scan(&row.AlarmID, &row.Name, &row.IP, &row.AlarmStatus, &row.Site, &row.ClearDate); err != nil {
			httpx.Internal(c, "failed to scan row: "+err.Error())
			return
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, "row iteration error: "+err.Error())
		return
	}

	var count int
	if err := db.DB.QueryRowContext(ctx, `
		SELECT COUNT(id)
		FROM public.device_down
		WHERE ($1::timestamp IS NULL OR clear_date >= $1)
		  AND ($2::timestamp IS NULL OR clear_date <= $2)
	`, startDate, endDate).Scan(&count); err != nil {
		httpx.Internal(c, "unable to fetch count: "+err.Error())
		return
	}

	httpx.OK(c, gin.H{
		"count":   count,
		"results": result,
	})
}

func GetAllLinkAlarmHistory(c *gin.Context) {
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

	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	var startDate *time.Time
	var endDate *time.Time

	if startDateStr != "" {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			httpx.BadRequest(c, "invalid startDate format. Use YYYY-MM-DD")
			return
		}
		startDate = &t
	}

	if endDateStr != "" {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			httpx.BadRequest(c, "invalid endDate format. Use YYYY-MM-DD")
			return
		}
		t = t.Add(24*time.Hour - time.Nanosecond)
		endDate = &t
	}

	type linkData struct {
		AlarmID     *string    `json:"alarm_id"`
		Hostname    *string    `json:"hostname"`
		IPAddress   *string    `json:"ipaddress"`
		Interface   *string    `json:"interface"`
		Site        *string    `json:"site"`
		AlarmStatus *string    `json:"alarm_status"`
		ClearDate   *time.Time `json:"clear_date"`
	}

	rows, err := db.DB.QueryContext(ctx, `
		SELECT alarm_id, hostname, ipaddress, interface, site, alarm_status, clear_date
		FROM public.link_down
		WHERE ($1::timestamp IS NULL OR clear_date >= $1)
		  AND ($2::timestamp IS NULL OR clear_date <= $2)
		ORDER BY clear_date
		LIMIT $3 OFFSET $4
	`, startDate, endDate, limit, offset)
	if err != nil {
		httpx.Internal(c, "database query failed: "+err.Error())
		return
	}
	defer rows.Close()

	var result []linkData
	for rows.Next() {
		var row linkData
		if err := rows.Scan(&row.AlarmID, &row.Hostname, &row.IPAddress, &row.Interface, &row.Site, &row.AlarmStatus, &row.ClearDate); err != nil {
			httpx.Internal(c, "failed to scan row: "+err.Error())
			return
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, "row iteration error: "+err.Error())
		return
	}

	var count int
	if err := db.DB.QueryRowContext(ctx, `
		SELECT COUNT(id)
		FROM public.link_down
		WHERE ($1::timestamp IS NULL OR clear_date >= $1)
		  AND ($2::timestamp IS NULL OR clear_date <= $2)
	`, startDate, endDate).Scan(&count); err != nil {
		httpx.Internal(c, "unable to fetch count: "+err.Error())
		return
	}

	httpx.OK(c, gin.H{
		"count":   count,
		"results": result,
	})
}