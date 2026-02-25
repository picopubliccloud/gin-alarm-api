package repository

import (
	"context"
	"time"

	db "github.com/picopubliccloud/alarm-api/internal/data/actual_data"
)

type DeviceData struct {
	AlarmID     *string    `json:"alarm_id"`
	Name        *string    `json:"name"`
	IP          *string    `json:"ip"`
	AlarmStatus *string    `json:"alarm_status"`
	Site        *string    `json:"site"`
	ClearDate   *time.Time `json:"clear_date"`
}

func AllDeviceAlarm(ctx context.Context,
	startDate *time.Time,
	endDate *time.Time,
	limit int,
	offset int,
) ([]DeviceData, int, error) {

	rows, err := db.DB.QueryContext(ctx, `select 
								alarm_id, 
								hostname, 
								ipaddress,
								alarm_status, 
								site, 
								clear_date 
							from public.device_down 
							WHERE ($1::timestamp IS NULL OR clear_date >= $1)
							AND   ($2::timestamp IS NULL OR clear_date <= $2)
							order by clear_date 
							limit $3 offset $4`, startDate, endDate, limit, offset)

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []DeviceData
	for rows.Next() {
		var row DeviceData
		if err := rows.Scan(&row.AlarmID, &row.Name, &row.IP, &row.AlarmStatus, &row.Site, &row.ClearDate); err != nil {
			return nil, 0, err
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var count int
	err = db.DB.QueryRowContext(ctx, `
					SELECT COUNT(id)
					FROM public.device_down
					WHERE ($1::timestamp IS NULL OR clear_date >= $1)
					AND   ($2::timestamp IS NULL OR clear_date <= $2)
				`, startDate, endDate).Scan(&count)

	if err != nil {
		return nil, 0, err
	}

	return result, count, nil
}
