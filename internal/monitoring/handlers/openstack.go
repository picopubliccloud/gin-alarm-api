package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/picopubliccloud/alarm-api/internal/data/actual_data"
	"github.com/lib/pq"
)

/* ============================================================
   Data Structures
============================================================ */

type OSIP struct {
	PublicIPID    string `json:"public_ip_id"`
	FloatingIP    string `json:"floating_ip"`
	Region        string `json:"region"`
	Status        string `json:"status"`
	InstanceName  string `json:"instance_name,omitempty"`
	PortID        string `json:"port_id,omitempty"`
	LastUpdatedAt string `json:"last_updated_at"`
}

type OSProject struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Enabled       bool     `json:"enabled"`
	Regions       []string `json:"regions"`
	TotalIPs      int      `json:"total_ips"`
	AssignedIPs   int      `json:"assigned_ips"`
	UnassignedIPs int      `json:"unassigned_ips"`
	IPs           []OSIP   `json:"ips"`
}

type OSPublicIP struct {
	ID        string    `json:"id"`
	Floating  string    `json:"floating_ip"`
	NetworkID string    `json:"network_id"`
	ProjectID string    `json:"project_id"`
	Region    string    `json:"region"`
	Status    string    `json:"status"` // ASSIGNED / UNASSIGNED
	Assigned  time.Time `json:"assigned_at"`
	Updated   time.Time `json:"last_updated"`
}

type IPOverview struct {
	TotalIPs      int `json:"total_ips"`
	AssignedIPs   int `json:"assigned_ips"`
	UnassignedIPs int `json:"unassigned_ips"`
	Projects      int `json:"projects"`
	Regions       int `json:"regions"`
}

/* ============================================================
   Helpers
============================================================ */

func getLatestSyncRunID(ctx context.Context) (int64, error) {
	var syncID int64
	err := db.DB.QueryRowContext(ctx, `
		SELECT id
		FROM public.sync_runs
		ORDER BY completed_at DESC
		LIMIT 1
	`).Scan(&syncID)
	return syncID, err
}

func abortIfCtxDone(c *gin.Context, err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		if !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{"error": "request timeout"})
		} else {
			c.Abort()
		}
		return true
	}
	return false
}

/* ============================================================
   Handlers
============================================================ */

// GET /api/openstack/projects
func GetOpenStackProjects(c *gin.Context) {
	ctx := c.Request.Context()

	syncID, err := getLatestSyncRunID(ctx)
	if err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := db.DB.QueryContext(ctx, `
		SELECT
			p.project_id,
			p.project_name,
			p.enabled,
			COALESCE(
				ARRAY_AGG(DISTINCT ip.region) FILTER (WHERE ip.region IS NOT NULL),
				'{}'
			) AS regions
		FROM public.projects p
		LEFT JOIN public.public_ip_assignments a
			ON p.project_id = a.project_id
		LEFT JOIN public.public_ips ip
			ON a.public_ip_id = ip.public_ip_id
			AND ip.sync_run_id = $1
		WHERE p.enabled = true
		GROUP BY p.project_id, p.project_name, p.enabled
		ORDER BY p.project_name;
	`, syncID)
	if err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var projects []OSProject
	for rows.Next() {
		if ctx.Err() != nil {
			abortIfCtxDone(c, ctx.Err())
			return
		}

		var p OSProject
		if err := rows.Scan(&p.ID, &p.Name, &p.Enabled, pq.Array(&p.Regions)); err != nil {
			continue
		}

		ipRows, err := db.DB.QueryContext(ctx, `
			SELECT
				ip.public_ip_id,
				ip.floating_ip,
				ip.region,
				a.status,
				a.port_id,
				a.last_updated_at
			FROM public.public_ip_assignments a
			JOIN public.public_ips ip ON a.public_ip_id = ip.public_ip_id
			WHERE a.project_id = $1
			  AND ip.sync_run_id = $2
		`, p.ID, syncID)

		if err != nil {
			if abortIfCtxDone(c, err) {
				return
			}
			p.IPs = []OSIP{}
			projects = append(projects, p)
			continue
		}

		// Important: close per-iteration (do NOT defer inside the loop)
		var ips []OSIP
		assigned := 0

		for ipRows.Next() {
			if ctx.Err() != nil {
				ipRows.Close()
				abortIfCtxDone(c, ctx.Err())
				return
			}

			var ip OSIP
			var lastUpdated pq.NullTime

			if err := ipRows.Scan(&ip.PublicIPID, &ip.FloatingIP, &ip.Region, &ip.Status, &ip.PortID, &lastUpdated); err != nil {
				continue
			}

			if lastUpdated.Valid {
				ip.LastUpdatedAt = lastUpdated.Time.Format(time.RFC3339)
			}
			if ip.Status == "ASSIGNED" {
				assigned++
			}
			ips = append(ips, ip)
		}
		ipRows.Close()

		p.IPs = ips
		p.TotalIPs = len(ips)
		p.AssignedIPs = assigned
		p.UnassignedIPs = len(ips) - assigned

		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if ctx.Err() != nil {
		abortIfCtxDone(c, ctx.Err())
		return
	}

	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

// GET /api/openstack/public-ips
func GetOpenStackPublicIPs(c *gin.Context) {
	ctx := c.Request.Context()

	syncID, err := getLatestSyncRunID(ctx)
	if err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := db.DB.QueryContext(ctx, `
		SELECT
			p.public_ip_id,
			p.floating_ip,
			p.external_network_id,
			a.project_id,
			p.region,
			COALESCE(a.status,'UNASSIGNED') AS status,
			a.assigned_at,
			a.last_updated_at
		FROM public.public_ips p
		LEFT JOIN public.public_ip_assignments a
			ON p.public_ip_id = a.public_ip_id
		WHERE p.sync_run_id = $1
	`, syncID)
	if err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var ips []OSPublicIP
	for rows.Next() {
		if ctx.Err() != nil {
			abortIfCtxDone(c, ctx.Err())
			return
		}

		var ip OSPublicIP
		if err := rows.Scan(&ip.ID, &ip.Floating, &ip.NetworkID, &ip.ProjectID, &ip.Region, &ip.Status, &ip.Assigned, &ip.Updated); err != nil {
			continue
		}
		ips = append(ips, ip)
	}

	if err := rows.Err(); err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if ctx.Err() != nil {
		abortIfCtxDone(c, ctx.Err())
		return
	}

	c.JSON(http.StatusOK, gin.H{"public_ips": ips})
}

// GET /api/openstack/overview
func GetOpenStackOverview(c *gin.Context) {
	ctx := c.Request.Context()

	syncID, err := getLatestSyncRunID(ctx)
	if err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var totalIPs, assignedIPs, projectsCount, regionsCount int

	err = db.DB.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total_ips,
			SUM(CASE WHEN a.status = 'ASSIGNED' THEN 1 ELSE 0 END) AS assigned_ips
		FROM public.public_ip_assignments a
		JOIN public.public_ips p
			ON a.public_ip_id = p.public_ip_id
		WHERE p.sync_run_id = $1
	`, syncID).Scan(&totalIPs, &assignedIPs)
	if err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = db.DB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM public.projects
		WHERE enabled = true
	`).Scan(&projectsCount)
	if err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = db.DB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT region)
		FROM public.public_ips
		WHERE sync_run_id = $1
	`, syncID).Scan(&regionsCount)
	if err != nil {
		if abortIfCtxDone(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if ctx.Err() != nil {
		abortIfCtxDone(c, ctx.Err())
		return
	}

	c.JSON(http.StatusOK, IPOverview{
		TotalIPs:      totalIPs,
		AssignedIPs:   assignedIPs,
		UnassignedIPs: totalIPs - assignedIPs,
		Projects:      projectsCount,
		Regions:       regionsCount,
	})
}