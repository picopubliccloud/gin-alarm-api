package handlers

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	db "github.com/picopubliccloud/alarm-api/internal/data/actual_data"
	"github.com/picopubliccloud/alarm-api/internal/httpx"
)

// GET /api/get-zone-districts
func GetZoneDistricts(c *gin.Context) {
	ctx := c.Request.Context()
	conn := db.DB

	rows, err := conn.QueryContext(ctx, `
		SELECT z.zonename, COUNT(d.ipaddress) AS count
		FROM public.zone_tbl z
		LEFT JOIN public.device_tbl d ON d.co_zone = z.zonename
		GROUP BY z.zonename
	`)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows.Close()

	zones := []gin.H{}

	for rows.Next() {
		var zoneName sql.NullString
		var count sql.NullInt64

		if err := rows.Scan(&zoneName, &count); err != nil {
			httpx.Internal(c, err.Error())
			return
		}

		z := gin.H{
			"districtsByZone": "",
			"count":           int64(0),
		}
		if zoneName.Valid {
			z["districtsByZone"] = zoneName.String
		}
		if count.Valid {
			z["count"] = count.Int64
		}

		zones = append(zones, z)
	}

	if err := rows.Err(); err != nil {
		httpx.Internal(c, "row iteration error: "+err.Error())
		return
	}

	httpx.OK(c, gin.H{
		"count":   len(zones),
		"results": zones,
	})
}