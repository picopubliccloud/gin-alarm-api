// internal/ticketing/jobs/partitions.go
package jobs

import (
	"context"
	"database/sql"
)

func EnsurePartitions(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		SELECT ops.create_ticket_updates_partition(date_trunc('month', now())::date);
		SELECT ops.create_ticket_updates_partition((date_trunc('month', now()) + interval '1 month')::date);

		SELECT ops.create_audit_events_partition(date_trunc('month', now())::date);
		SELECT ops.create_audit_events_partition((date_trunc('month', now()) + interval '1 month')::date);
	`)
	return err
}