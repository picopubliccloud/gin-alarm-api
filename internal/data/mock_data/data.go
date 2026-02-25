package data

import (
	"time"
	"github.com/picopubliccloud/alarm-api/internal/monitoring/models"
)
var SiteSummaries = []models.SiteSummary{
	{
		Site:  "Kaliakoir",
		Total: 103,
		Up:    101,
		Down:  2,
		DownDevices: []models.DownDevice{
			{Name: "Nokia 7220 IXR-H4 (Spine Switch)", IP: "192.168.16.200", Site: "Kaliakoir", DownSince: time.Now().Add(-43 * time.Minute)},
			{Name: "PowerEdge R7615 (Storage Node)", IP: "192.168.10.100", Site: "Kaliakoir", DownSince: time.Now().Add(-2 * time.Hour)},
		},
	},
	{Site: "Jashore", Total: 74, Up: 74, Down: 0},
	{Site: "NMC", Total: 2, Up: 2, Down: 0},
}

var PSUHealth = []models.PSUHealth{
	{Site: "Kaliakoir", Total: 103, Redundant: 0, Failed: 0},
	{Site: "Jashore", Total: 74, Redundant: 0, Failed: 0},
	{Site: "NMC", Total: 2, Redundant: 0, Failed: 0},
}

var AssetDistribution = []models.AssetDistribution{
	{Name: "Switches", Value: 58},
	{Name: "Routers", Value: 4},
	{Name: "Servers", Value: 106},
	{Name: "Security Devices", Value: 7},
}