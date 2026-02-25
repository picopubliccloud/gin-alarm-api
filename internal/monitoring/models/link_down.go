package models

import "time"

type LinkDown struct {
	ID              int        `json:"id"`
	AlarmID         string     `json:"alarm_id"`
	AlarmName       string     `json:"alarm_name"`
	Hostname        string     `json:"hostname"`
	IPAddress       string     `json:"ipaddress"`
	Interface       string     `json:"interface"`
	LinkType        string     `json:"link_type"`
	Description     *string    `json:"description"`
	LinkStatus      string     `json:"link_status"`
	AlarmStatus     string     `json:"alarm_status"`
	Severity        string     `json:"severity"`
	CreateDate      time.Time  `json:"create_date"`
	ClearDate       *time.Time `json:"clear_date"`
	AcknowledgeDate *time.Time `json:"acknowledge_date"`
	LastSyncDate    *time.Time `json:"last_sync_date"`
	Comments        string     `json:"comments"`
	AcknowledgeBy   string     `json:"acknowledge_by"`
	ClearBy         string     `json:"clear_by"`
	Duration        *int       `json:"duration"`
	FlapCount       *int       `json:"flap_count"`
	Site            *string    `json:"site"`
}
