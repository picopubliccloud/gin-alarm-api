package handlers

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	db "github.com/picopubliccloud/alarm-api/internal/data/actual_data"
	"github.com/picopubliccloud/alarm-api/internal/httpx"
)

// GET /api/get-nodes-edges/:district
func GetNodesEdges(c *gin.Context) {
	ctx := c.Request.Context()

	district := c.Param("district")
	if district == "" {
		httpx.BadRequest(c, "district parameter is required")
		return
	}

	conn := db.DB

	// 1) Fetch IP addresses for the district
	rows, err := conn.QueryContext(ctx, `
		SELECT ipaddress
		FROM public.device_tbl
		WHERE co_zone = $1
	`, district)
	if err != nil {
		httpx.Internal(c, err.Error())
		return
	}
	defer rows.Close()

	var ipAddresses []string
	for rows.Next() {
		var ip sql.NullString
		if err := rows.Scan(&ip); err != nil {
			httpx.Internal(c, err.Error())
			return
		}
		if ip.Valid && ip.String != "" {
			ipAddresses = append(ipAddresses, ip.String)
		}
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(c, err.Error())
		return
	}

	results := make([]gin.H, 0, len(ipAddresses))

	// 2) For each IP, fetch topology links
	for _, ip := range ipAddresses {
		nodes := make(map[string]gin.H)
		edges := []gin.H{}

		topologyRows, err := conn.QueryContext(ctx, `
			SELECT local_ipaddress, local_hostname,
			       neighbor_ipaddress, neighbor_hostname,
			       local_intf, neighbor_intf, description
			FROM public.network_topology_link_tbl
			WHERE local_ipaddress = $1
		`, ip)
		if err != nil {
			// Previously you "continue"d silently; better to keep behavior but include a minimal hint.
			// If you want strict behavior, replace this with httpx.Internal + return.
			results = append(results, gin.H{
				"ip":    ip,
				"nodes": []gin.H{},
				"edges": []gin.H{},
				"error": "failed to query topology for ip",
			})
			continue
		}

		for topologyRows.Next() {
			var localIP, localHost, neighborIP, neighborHost, localIntf, neighborIntf, desc sql.NullString
			if err := topologyRows.Scan(&localIP, &localHost, &neighborIP, &neighborHost, &localIntf, &neighborIntf, &desc); err != nil {
				topologyRows.Close()
				httpx.Internal(c, err.Error())
				return
			}

			if localIP.Valid && localHost.Valid {
				if _, exists := nodes[localIP.String]; !exists {
					nodes[localIP.String] = gin.H{"ip": localIP.String, "hostname": localHost.String}
				}
			}
			if neighborIP.Valid && neighborHost.Valid {
				if _, exists := nodes[neighborIP.String]; !exists {
					nodes[neighborIP.String] = gin.H{"ip": neighborIP.String, "hostname": neighborHost.String}
				}
			}

			edges = append(edges, gin.H{
				"local_hostname":     localHost.String,
				"local_ipaddress":    localIP.String,
				"local_intf":         localIntf.String,
				"neighbor_hostname":  neighborHost.String,
				"neighbor_ipaddress": neighborIP.String,
				"neighbor_intf":      neighborIntf.String,
				"description":        desc.String,
			})
		}

		if err := topologyRows.Err(); err != nil {
			topologyRows.Close()
			httpx.Internal(c, err.Error())
			return
		}
		topologyRows.Close()

		results = append(results, gin.H{
			"ip":    ip,
			"nodes": nodesToList(nodes),
			"edges": edges,
		})
	}

	httpx.OK(c, gin.H{
		"count":   len(results),
		"results": results,
	})
}

// Convert nodes map to list
func nodesToList(nodes map[string]gin.H) []gin.H {
	list := make([]gin.H, 0, len(nodes))
	for _, v := range nodes {
		list = append(list, v)
	}
	return list
}