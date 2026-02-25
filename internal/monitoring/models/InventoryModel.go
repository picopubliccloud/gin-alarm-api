package models

import "time"

type InventoryModel struct {
	ID                       int        `json:"id"`
	AssetType                *string    `json:"asset_type"`
	AssetTag                 *string    `json:"asset_tag"`
	BrandModel               *string    `json:"brand_model"`
	NicLineCard              *string    `json:"nic_line_card"`
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
	RemovedDate              *time.Time `json:"removed_date"`
	RemovedBy                *string    `json:"removed_by"`
	IsActive                 bool       `json:"is_active"`
	Purpose                  *string    `json:"purpose"`
	Verification             *string    `json:"verification"`
	UpDownStatus             *string    `json:"up_down_status"`
	LastDownTime             *time.Time `json:"last_down_time"`
	LastUpTime               *time.Time `json:"last_up_time"`
	LastDownCheckTime        *time.Time `json:"last_down_check_time"`
	TotalPowerSupplyCount    *string    `json:"total_power_supply_count"`
	UpPowerSupplyCount       *string    `json:"up_power_supply_count"`
	DownPowerSupplyCount     *string    `json:"down_power_supply_count"`
	LastPowerSupplyCheckTime *time.Time `json:"last_power_supply_check_time"`
}
