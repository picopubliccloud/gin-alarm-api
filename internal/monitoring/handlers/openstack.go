package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/picopubliccloud/alarm-api/internal/data/actual_data"
	"github.com/lib/pq"
)

// ----------------------------
// Data Structures
// ----------------------------
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

// ----------------------------
// Helpers
// ----------------------------
func getLatestSyncRunID() (int64, error) {
	var syncID int64
	err := db.DB.QueryRow(`SELECT id FROM public.sync_runs ORDER BY completed_at DESC LIMIT 1`).Scan(&syncID)
	return syncID, err
}

// ----------------------------
// Handlers
// ----------------------------

// GET /api/openstack/projects
func GetOpenStackProjects(c *gin.Context) {
	syncID, err := getLatestSyncRunID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := db.DB.Query(`
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var projects []OSProject
	for rows.Next() {
		var p OSProject
		if err := rows.Scan(&p.ID, &p.Name, &p.Enabled, pq.Array(&p.Regions)); err != nil {
			continue
		}

		// Fetch current IPs for this project
		ipRows, err := db.DB.Query(`
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
			p.IPs = []OSIP{}
		} else {
			defer ipRows.Close()
			var ips []OSIP
			assigned := 0
			for ipRows.Next() {
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
			p.IPs = ips
			p.TotalIPs = len(ips)
			p.AssignedIPs = assigned
			p.UnassignedIPs = len(ips) - assigned
		}

		projects = append(projects, p)
	}

	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

// GET /api/openstack/public-ips
func GetOpenStackPublicIPs(c *gin.Context) {
	syncID, err := getLatestSyncRunID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := db.DB.Query(`
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var ips []OSPublicIP
	for rows.Next() {
		var ip OSPublicIP
		if err := rows.Scan(&ip.ID, &ip.Floating, &ip.NetworkID, &ip.ProjectID, &ip.Region, &ip.Status, &ip.Assigned, &ip.Updated); err != nil {
			continue
		}
		ips = append(ips, ip)
	}

	c.JSON(http.StatusOK, gin.H{"public_ips": ips})
}

// GET /api/openstack/overview
func GetOpenStackOverview(c *gin.Context) {
	syncID, err := getLatestSyncRunID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var totalIPs, assignedIPs, unassignedIPs, projectsCount, regionsCount int

	// Total IPs and assigned/unassigned (active only)
	err = db.DB.QueryRow(`
		SELECT
			COUNT(*) AS total_ips,
			SUM(CASE WHEN a.status = 'ASSIGNED' THEN 1 ELSE 0 END) AS assigned_ips
		FROM public.public_ip_assignments a
		JOIN public.public_ips p
			ON a.public_ip_id = p.public_ip_id
		WHERE p.sync_run_id = $1
	`, syncID).Scan(&totalIPs, &assignedIPs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	unassignedIPs = totalIPs - assignedIPs

	// Projects count (enabled only)
	err = db.DB.QueryRow(`SELECT COUNT(*) FROM public.projects WHERE enabled = true`).Scan(&projectsCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Regions count (active IPs only)
	err = db.DB.QueryRow(`
		SELECT COUNT(DISTINCT region)
		FROM public.public_ips
		WHERE sync_run_id = $1
	`, syncID).Scan(&regionsCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, IPOverview{
		TotalIPs:      totalIPs,
		AssignedIPs:   assignedIPs,
		UnassignedIPs: unassignedIPs,
		Projects:      projectsCount,
		Regions:       regionsCount,
	})
}
