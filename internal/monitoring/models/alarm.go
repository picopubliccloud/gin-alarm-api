package models

import "time"

type DownDevice struct {
	AlarmID         string     `json:"alarm_id"`
	Name            string     `json:"name"` // Hostname
	IP              string     `json:"ip"`
	AlarmStatus     string     `json:"alarm_status"`
	Severity        string     `json:"severity"`
	DownSince       time.Time  `json:"down_since"`
	Site            string     `json:"site"`
	Comments        string     `json:"comments"`
	ClearedBy       string     `json:"clear_by"`
	AcknowledgeBy   string     `json:"acknowledge_by"`
	ClearDate       *time.Time `json:"clear_date"`
	AcknowledgeDate *time.Time `json:"acknowledge_date"`
}

type SiteSummary struct {
	Site        string       `json:"site"`
	Total       int          `json:"total"`
	Up          int          `json:"up"`
	Down        int          `json:"down"`
	DownDevices []DownDevice `json:"down_devices"`
}

type PSUHealth struct {
	Site      string `json:"site"`
	Total     int    `json:"total"`
	Redundant int    `json:"redundant"`
	Failed    int    `json:"failed"`
}

type AssetDistribution struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// Device represents a full inventory item
type Device struct {
	ID                       int        `json:"id"`
	AssetType                *string    `json:"asset_type"`
	AssetTag                 *string    `json:"asset_tag"`
	BrandModel               *string    `json:"brand_model"`
	NICLineCard              *string    `json:"nic_line_card"`
	PSU                      *string    `json:"psu"`
	Site                     *string    `json:"site"`
	Rack                     *string    `json:"rack"`
	Unit                     *string    `json:"unit"`
	Owner                    *string    `json:"owner"`
	Status                   *string    `json:"status"`
	Remark                   *string    `json:"remark"`
	MgmtIPAddress            *string    `json:"mgmt_ip_address"`
	SecondaryIP              *string    `json:"secondary_ip"`
	HostName                 *string    `json:"host_name"`
	OS                       *string    `json:"os"`
	OSVersion                *string    `json:"os_version"`
	AddedDate                *time.Time `json:"added_date"`
	AddedBy                  *string    `json:"added_by"`
	LastModifiedDate         *time.Time `json:"last_modified_date"`
	LastModifiedBy           *string    `json:"last_modified_by"`
	RemovedDate              *time.Time `json:"removed_date"` // nullable
	RemovedBy                *string    `json:"removed_by"`
	IsActive                 *bool      `json:"is_active"`
	Purpose                  *string    `json:"purpose"`
	Verification             *string    `json:"verification"`
	UpDownStatus             *string    `json:"up_down_status"`
	LastDownTime             *time.Time `json:"last_down_time"`       // nullable
	LastUpTime               *time.Time `json:"last_up_time"`         // nullable
	LastDownCheckTime        *time.Time `json:"last_down_check_time"` // nullable
	TotalPowerSupplyCount    *int       `json:"total_power_supply_count"`
	UpPowerSupplyCount       *int       `json:"up_power_supply_count"`
	DownPowerSupplyCount     *int       `json:"down_power_supply_count"`
	LastPowerSupplyCheckTime *time.Time `json:"last_power_supply_check_time"` // nullable
}
