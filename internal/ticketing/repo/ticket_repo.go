package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
)

type TicketRepo struct {
	db         *sql.DB
	updateRepo *UpdateRepo
}

// func NewTicketRepo(db *sql.DB) *TicketRepo { return &TicketRepo{db: db} }
func NewTicketRepo(db *sql.DB, ur *UpdateRepo) *TicketRepo {
	return &TicketRepo{db: db, updateRepo: ur}
}

// Cursor pagination: cursor is ticket_number (int64). NextCursor is last ticket_number in page.
func (r *TicketRepo) List(ctx context.Context, filter TicketListFilter) ([]models.TicketListItem, string, bool, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	conds := []string{"1=1"}
	args := []any{}
	argn := 1

	// RBAC: restrict projects for customer
	if len(filter.AllowedProjectIDs) > 0 {
		conds = append(conds, fmt.Sprintf("h.project_id = ANY($%d)", argn))
		args = append(args, filter.AllowedProjectIDs)
		argn++
	}

	// Visibility: max visibility
	if filter.MaxVisibility == models.VisibilityPublic {
		conds = append(conds, fmt.Sprintf("h.visibility = $%d", argn))
		args = append(args, string(models.VisibilityPublic))
		argn++
	} else if filter.MaxVisibility == models.VisibilityInternal {
		conds = append(conds, fmt.Sprintf("h.visibility IN ($%d,$%d)", argn, argn+1))
		args = append(args, string(models.VisibilityPublic), string(models.VisibilityInternal))
		argn += 2
	} else {
		// RESTRICTED allowed -> everything
	}

	// Optional: status / severity filters
	if filter.Status != "" {
		conds = append(conds, fmt.Sprintf("h.status = $%d", argn))
		args = append(args, string(filter.Status))
		argn++
	}
	if filter.Severity != "" {
		conds = append(conds, fmt.Sprintf("h.severity = $%d", argn))
		args = append(args, string(filter.Severity))
		argn++
	}

	// Cursor (descending by ticket_number)
	if filter.CursorTicketNumber > 0 {
		conds = append(conds, fmt.Sprintf("h.ticket_number < $%d", argn))
		args = append(args, filter.CursorTicketNumber)
		argn++
	}

	// Pool filter: no active OWNER assignment
	if filter.OnlyPool {
		conds = append(conds, `
NOT EXISTS (
  SELECT 1 FROM ops.ticket_assignments a
  WHERE a.ticket_id = h.ticket_id
    AND a.assignment_role = 'OWNER'
    AND a.unassigned_at IS NULL
)`)
	}

	// ✅ UPDATED: add has_owner + is_locked columns
	q := fmt.Sprintf(`
SELECT
  h.ticket_id::text,
  h.ticket_number,
  h.project_id,
  p.project_name,
  h.customer_id::text,
  h.service_id::text,
  h.ticket_type::text,
  h.status::text,
  h.severity::text,
  h.priority_score,
  h.visibility::text,
  h.component_id,
  h.is_known_issue,
  h.created_at,
  h.updated_at,
  tx.title,

  -- owner info (null if pool)
  ow.owner_user_id,
  ow.owner_display_name,
  ow.owner_assigned_at,

  -- has_owner: active OWNER exists
  EXISTS (
    SELECT 1
    FROM ops.ticket_assignments a
    WHERE a.ticket_id = h.ticket_id
      AND a.assignment_role = 'OWNER'
      AND a.unassigned_at IS NULL
  ) AS has_owner,

  -- is_locked: lock exists and not expired
  EXISTS (
    SELECT 1
    FROM ops.ticket_locks l
    WHERE l.ticket_id = h.ticket_id
      AND l.lock_expires_at > now()
  ) AS is_locked

FROM ops.tickets_header h
JOIN ops.tickets_text tx ON tx.ticket_id = h.ticket_id
LEFT JOIN ops.projects p ON p.project_id = h.project_id

LEFT JOIN LATERAL (
  SELECT
    a.user_id::text AS owner_user_id,
    COALESCE(u.display_name, u.email::text, u.external_subject) AS owner_display_name,
    a.assigned_at AS owner_assigned_at
  FROM ops.ticket_assignments a
  LEFT JOIN ops.users u ON u.user_id = a.user_id
  WHERE a.ticket_id = h.ticket_id
    AND a.assignment_role = 'OWNER'
    AND a.unassigned_at IS NULL
  ORDER BY a.assigned_at DESC
  LIMIT 1
) ow ON true

WHERE %s
ORDER BY h.ticket_number DESC
LIMIT %d;
`, strings.Join(conds, " AND "), limit+1) // +1 to detect hasMore

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	items := make([]models.TicketListItem, 0, limit+1)
	for rows.Next() {
		var it models.TicketListItem
		var tt, st, sv, vis string

		var ownerUserID sql.NullString
		var ownerDisplayName sql.NullString
		var ownerAssignedAt sql.NullTime

		// ✅ UPDATED: scan has_owner + is_locked
		if err := rows.Scan(
			&it.TicketID,
			&it.TicketNumber,
			&it.ProjectID,
			&it.ProjectName,
			&it.CustomerID,
			&it.ServiceID,
			&tt,
			&st,
			&sv,
			&it.Priority,
			&vis,
			&it.ComponentID,
			&it.IsKnownIssue,
			&it.CreatedAt,
			&it.UpdatedAt,
			&it.Title,

			&ownerUserID,
			&ownerDisplayName,
			&ownerAssignedAt,

			&it.HasOwner,
			&it.IsLocked,
		); err != nil {
			return nil, "", false, err
		}



		it.TicketType = models.TicketType(tt)
		it.Status = models.TicketStatus(st)
		it.Severity = models.TicketSeverity(sv)
		it.Visibility = models.TicketVisibility(vis)

		if ownerUserID.Valid {
			s := ownerUserID.String
			it.OwnerUserID = &s
		}
		if ownerDisplayName.Valid {
			s := ownerDisplayName.String
			it.OwnerDisplayName = &s
		}
		if ownerAssignedAt.Valid {
			t := ownerAssignedAt.Time
			it.OwnerAssignedAt = &t
		}

		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, err
	}

	hasMore := false
	if len(items) > limit {
		hasMore = true
		items = items[:limit]
	}

	nextCursor := ""
	if len(items) > 0 && hasMore {
		nextCursor = fmt.Sprintf("%d", items[len(items)-1].TicketNumber)
	}

	return items, nextCursor, hasMore, nil
}

func (r *TicketRepo) Get(
	ctx context.Context,
	ticketID string,
	allowedProjectIDs []string,
	maxVis models.TicketVisibility,
) (*models.TicketDetail, error) {

	conds := []string{"h.ticket_id = $1"}
	args := []any{ticketID}
	argn := 2

	if len(allowedProjectIDs) > 0 {
		conds = append(conds, fmt.Sprintf("h.project_id = ANY($%d)", argn))
		args = append(args, allowedProjectIDs)
		argn++
	}

	if maxVis == models.VisibilityPublic {
		conds = append(conds, fmt.Sprintf("h.visibility = $%d", argn))
		args = append(args, string(models.VisibilityPublic))
		argn++
	} else if maxVis == models.VisibilityInternal {
		conds = append(conds, fmt.Sprintf("h.visibility IN ($%d,$%d)", argn, argn+1))
		args = append(args, string(models.VisibilityPublic), string(models.VisibilityInternal))
		argn += 2
	}

	q := fmt.Sprintf(`
SELECT
  h.ticket_id::text,
  h.ticket_number,
  h.project_id,
  h.customer_id::text,
  h.service_id::text,

  h.ticket_type::text,
  h.status::text,
  h.severity::text,
  h.priority_score,

  h.visibility::text,
  h.component_id,
  h.is_known_issue,

  u_created.display_name,
  h.created_by_actor::text,

  h.master_ticket_id::text,
  h.problem_id::text,

  h.created_at,
  h.updated_at,
  h.closed_at,
  u_closed.display_name,
  h.resolution_code::text,

  tx.title,
  tx.description,
  tx.impact_summary,
  tx.suspected_cause,
  tx.created_at,

  ow.owner_user_id,
  ow.owner_assigned_by,
  ow.owner_assigned_at,

  lk.locked_by,
  lk.locked_at,
  lk.lock_expires_at,
  lk.lock_reason_id

FROM ops.tickets_header h
JOIN ops.tickets_text tx ON tx.ticket_id = h.ticket_id

LEFT JOIN ops.users u_created ON u_created.user_id = h.created_by
LEFT JOIN ops.users u_closed  ON u_closed.user_id  = h.closed_by

LEFT JOIN LATERAL (
  SELECT
    a.user_id::text      AS owner_user_id,
    a.assigned_by::text  AS owner_assigned_by,
    a.assigned_at        AS owner_assigned_at
  FROM ops.ticket_assignments a
  WHERE a.ticket_id = h.ticket_id
    AND a.assignment_role = 'OWNER'
    AND a.unassigned_at IS NULL
  ORDER BY a.assigned_at DESC
  LIMIT 1
) ow ON true

LEFT JOIN LATERAL (
  SELECT
    l.locked_by::text    AS locked_by,
    l.locked_at          AS locked_at,
    l.lock_expires_at    AS lock_expires_at,
    l.lock_reason_id     AS lock_reason_id
  FROM ops.ticket_locks l
  WHERE l.ticket_id = h.ticket_id
    AND l.lock_expires_at > now()
  LIMIT 1
) lk ON true

WHERE %s;
`, strings.Join(conds, " AND "))

	var d models.TicketDetail
	var tt, st, sv, vis, cba string

	var master, prob, resCode sql.NullString
	var desc, impact, cause sql.NullString

	var closedAtTs sql.NullTime
	var closedByName sql.NullString
	var createdByName sql.NullString

	var ownerUserID, ownerAssignedBy sql.NullString
	var ownerAssignedAt sql.NullTime

	var lockedBy sql.NullString
	var lockedAt, lockExpiresAt sql.NullTime
	var lockReasonID sql.NullInt16

	err := r.db.QueryRowContext(ctx, q, args...).Scan(
		&d.Header.TicketID,
		&d.Header.TicketNumber,
		&d.Header.ProjectID,
		&d.Header.CustomerID,
		&d.Header.ServiceID,

		&tt,
		&st,
		&sv,
		&d.Header.Priority,

		&vis,
		&d.Header.ComponentID,
		&d.Header.IsKnownIssue,

		&createdByName,
		&cba,

		&master,
		&prob,

		&d.Header.CreatedAt,
		&d.Header.UpdatedAt,
		&closedAtTs,
		&closedByName,
		&resCode,

		&d.Text.Title,
		&desc,
		&impact,
		&cause,
		&d.Text.CreatedAt,

		&ownerUserID,
		&ownerAssignedBy,
		&ownerAssignedAt,

		&lockedBy,
		&lockedAt,
		&lockExpiresAt,
		&lockReasonID,
	)
	if err != nil {
		return nil, err
	}

	d.Header.TicketType = models.TicketType(tt)
	d.Header.Status = models.TicketStatus(st)
	d.Header.Severity = models.TicketSeverity(sv)
	d.Header.Visibility = models.TicketVisibility(vis)
	d.Header.CreatedByActor = models.ActorType(cba)

	if createdByName.Valid {
		d.Header.CreatedBy = createdByName.String
	}

	if master.Valid {
		s := master.String
		d.Header.MasterTicketID = &s
	}
	if prob.Valid {
		s := prob.String
		d.Header.ProblemID = &s
	}
	if closedAtTs.Valid {
		t := closedAtTs.Time
		d.Header.ClosedAt = &t
	}
	if closedByName.Valid {
		s := closedByName.String
		d.Header.ClosedBy = &s
	}
	if resCode.Valid {
		rc := models.ResolutionCode(resCode.String)
		d.Header.ResolutionCode = &rc
	}

	d.Text.TicketID = d.Header.TicketID

	if desc.Valid {
		s := desc.String
		d.Text.Description = &s
	}
	if impact.Valid {
		s := impact.String
		d.Text.ImpactSummary = &s
	}
	if cause.Valid {
		s := cause.String
		d.Text.SuspectedCause = &s
	}

	// Owner view
	if ownerUserID.Valid {
		uid := ownerUserID.String

		var ab *string
		if ownerAssignedBy.Valid {
			s := ownerAssignedBy.String
			ab = &s
		}

		var at *time.Time
		if ownerAssignedAt.Valid {
			t := ownerAssignedAt.Time
			at = &t
		}

		d.Owner = &models.TicketOwnerView{
			UserID:     &uid,
			AssignedBy: ab,
			AssignedAt: at,
		}
	}

	// Lock view
	if lockedBy.Valid {
		lb := lockedBy.String

		var la *time.Time
		if lockedAt.Valid {
			t := lockedAt.Time
			la = &t
		}

		var le *time.Time
		if lockExpiresAt.Valid {
			t := lockExpiresAt.Time
			le = &t
		}

		var lr *int16
		if lockReasonID.Valid {
			v := lockReasonID.Int16
			lr = &v
		}

		d.Lock = &models.TicketLockView{
			Locked:        true,
			LockedBy:      &lb,
			LockedAt:      la,
			LockExpiresAt: le,
			LockReasonID:  lr,
		}
	} else {
		d.Lock = &models.TicketLockView{Locked: false}
	}

	updates, err := r.updateRepo.ListByTicket(ctx, ticketID, maxVis, 100)
	if err != nil {
		return nil, err
	}
	d.Updates = updates

	return &d, nil
}

func (r *TicketRepo) Create(ctx context.Context, tx *sql.Tx, h *models.TicketHeader, t *models.TicketText) (string, int64, error) {
	const qh = `
INSERT INTO ops.tickets_header (
  project_id, customer_id, service_id,
  ticket_type, status, severity, priority_score,
  visibility, component_id, is_known_issue,
  created_by, created_by_actor,
  created_at, updated_at
)
VALUES ($1,$2,$3,$4,'NEW',$5,0,$6,$7,false,$8,$9,now(),now())
RETURNING ticket_id::text, ticket_number;
`
	var ticketID string
	var ticketNumber int64
	if err := tx.QueryRowContext(ctx, qh,
		h.ProjectID,
		h.CustomerID,
		h.ServiceID,
		string(h.TicketType),
		string(h.Severity),
		string(h.Visibility),
		h.ComponentID,
		h.CreatedBy,
		string(h.CreatedByActor),
	).Scan(&ticketID, &ticketNumber); err != nil {
		return "", 0, err
	}

	const qt = `
INSERT INTO ops.tickets_text (ticket_id, title, description, impact_summary, suspected_cause)
VALUES ($1,$2,$3,$4,$5);
`
	if _, err := tx.ExecContext(ctx, qt, ticketID, t.Title, t.Description, t.ImpactSummary, t.SuspectedCause); err != nil {
		return "", 0, err
	}
	return ticketID, ticketNumber, nil
}

type TicketListFilter struct {
	AllowedProjectIDs  []string
	MaxVisibility      models.TicketVisibility
	Status             models.TicketStatus
	Severity           models.TicketSeverity
	CursorTicketNumber int64
	Limit              int
	OnlyPool           bool
}

// --- Compatibility wrappers for older service code ---

func (r *TicketRepo) InsertTicket(ctx context.Context, tx *sql.Tx, h *models.TicketHeader, t *models.TicketText) (string, int64, error) {
	return r.Create(ctx, tx, h, t)
}

func (r *TicketRepo) ListTickets(ctx context.Context, filter TicketListFilter) ([]models.TicketListItem, string, bool, error) {
	return r.List(ctx, filter)
}

func (r *TicketRepo) GetTicketFull(ctx context.Context, ticketID string, allowedProjectIDs []string, maxVis models.TicketVisibility) (*models.TicketDetail, error) {
	return r.Get(ctx, ticketID, allowedProjectIDs, maxVis)
}

// Close is used by closure_service.
func (r *TicketRepo) Close(ctx context.Context, tx *sql.Tx, ticketID string, resolutionCode models.ResolutionCode, closedBy string, closedAt time.Time) error {
	const q = `
UPDATE ops.tickets_header
SET status='CLOSED',
    resolution_code=$2,
    closed_by=$3::uuid,
    closed_at=$4,
    updated_at=now()
WHERE ticket_id=$1::uuid;
`
	_, err := tx.ExecContext(ctx, q, ticketID, string(resolutionCode), closedBy, closedAt)
	return err
}

// Reopen: sets status back to IN_PROGRESS and increments reopened_count.
func (r *TicketRepo) Reopen(ctx context.Context, tx *sql.Tx, ticketID string) error {
	const q = `
UPDATE ops.tickets_header
SET status='IN_PROGRESS',
    reopened_count = reopened_count + 1,
    updated_at=now()
WHERE ticket_id=$1::uuid AND status='CLOSED';
`
	_, err := tx.ExecContext(ctx, q, ticketID)
	return err
}


func (r *TicketRepo) UpdateStatus(ctx context.Context, tx *sql.Tx, ticketID string, newStatus models.TicketStatus, actorUserID string) error {
    if ticketID == "" {
        return fmt.Errorf("ticket_id required")
    }
    if actorUserID == "" {
        return fmt.Errorf("actor_user_id required")
    }

    // Optional: validate current status + transition
    var cur string
    err := tx.QueryRowContext(ctx, `SELECT status::text FROM ops.tickets_header WHERE ticket_id=$1::uuid`, ticketID).Scan(&cur)
    if err != nil {
        return err
    }
    curSt := models.TicketStatus(cur)

    if err := validateStatusTransition(curSt, newStatus); err != nil {
        return err
    }

    // Do not allow changing status after CLOSED
    if strings.EqualFold(string(curSt), "CLOSED") {
        return fmt.Errorf("ticket is CLOSED; status cannot be changed")
    }

    _, err = tx.ExecContext(ctx, `
UPDATE ops.tickets_header
SET status = $2::ops.ticket_status,
    updated_at = now()
WHERE ticket_id = $1::uuid
`, ticketID, string(newStatus))

    return err
}

func validateStatusTransition(from, to models.TicketStatus) error {
    f := strings.ToUpper(string(from))
    t := strings.ToUpper(string(to))
    if f == t {
        return nil
    }

    // Simple recommended rules (adjust as you want)
    allowed := map[string]map[string]bool{
        "NEW": {
            "ACKNOWLEDGED": true,
            "IN_PROGRESS":  true,
            "WAITING_CUSTOMER": true,
            "WAITING_VENDOR":   true,
        },
        "ACKNOWLEDGED": {
            "IN_PROGRESS": true,
            "WAITING_CUSTOMER": true,
            "WAITING_VENDOR":   true,
        },
        "IN_PROGRESS": {
            "WAITING_CUSTOMER": true,
            "WAITING_VENDOR":   true,
            "MITIGATED":        true,
            "RESOLVED":         true,
        },
        "WAITING_CUSTOMER": {
            "IN_PROGRESS": true,
            "RESOLVED":    true,
        },
        "WAITING_VENDOR": {
            "IN_PROGRESS": true,
            "RESOLVED":    true,
        },
        "MITIGATED": {
            "IN_PROGRESS": true,
            "RESOLVED":    true,
        },
        "RESOLVED": {
            // Usually you go to CLOSED via Close endpoint (enforce closure summary)
            "IN_PROGRESS": true, // reopen case
        },
        "CLOSED": {},
    }

    if allowed[f] != nil && allowed[f][t] {
        return nil
    }
    return fmt.Errorf("invalid status transition: %s -> %s", f, t)
}

type TicketSummaryFilter struct {
	OnlyPool      bool
	Status        string
	Severity      string
	IncludeClosed bool

	AllowedProjects []string
	MaxVis          models.TicketVisibility
}
func (r *TicketRepo) Summary(ctx context.Context, f TicketSummaryFilter) (*models.TicketsSummaryResponse, error) {
	conds := []string{"1=1"}
	args := []any{}
	argn := 1

	// ✅ RBAC: project restriction
	if len(f.AllowedProjects) > 0 {
		conds = append(conds, fmt.Sprintf("h.project_id = ANY($%d)", argn))
		args = append(args, f.AllowedProjects)
		argn++
	}

	// ✅ RBAC: visibility restriction (same logic you use in Get/List)
	// If your visibility is enum text:
	if f.MaxVis == models.VisibilityPublic {
		conds = append(conds, fmt.Sprintf("h.visibility = $%d", argn))
		args = append(args, string(models.VisibilityPublic))
		argn++
	} else if f.MaxVis == models.VisibilityInternal {
		conds = append(conds, fmt.Sprintf("h.visibility IN ($%d,$%d)", argn, argn+1))
		args = append(args, string(models.VisibilityPublic), string(models.VisibilityInternal))
		argn += 2
	}

	// filters
	if f.Status != "" {
		conds = append(conds, fmt.Sprintf("h.status = $%d", argn))
		args = append(args, f.Status)
		argn++
	}

	if f.Severity != "" {
		conds = append(conds, fmt.Sprintf("h.severity = $%d", argn))
		args = append(args, f.Severity)
		argn++
	}

	if !f.IncludeClosed {
		conds = append(conds, "h.status <> 'CLOSED'")
	}

	// pool filter: “no active owner”
	// NOTE: this must match your real pool definition
	if f.OnlyPool {
		conds = append(conds, `
NOT EXISTS (
  SELECT 1
  FROM ops.ticket_assignments a
  WHERE a.ticket_id = h.ticket_id
    AND a.assignment_role = 'OWNER'::ops.assignment_role
    AND a.unassigned_at IS NULL
)`)
	}

	where := strings.Join(conds, " AND ")

	// --- totals
	var total int64
	qTotal := "SELECT COUNT(*) FROM ops.tickets_header h WHERE " + where
	if err := r.db.QueryRowContext(ctx, qTotal, args...).Scan(&total); err != nil {
		return nil, err
	}

	// --- by status
	byStatus := map[string]int64{}
	qStatus := `
SELECT h.status, COUNT(*)
FROM ops.tickets_header h
WHERE ` + where + `
GROUP BY h.status
`
	rows, err := r.db.QueryContext(ctx, qStatus, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var k string
		var v int64
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		byStatus[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// --- by severity
	bySev := map[string]int64{}
	qSev := `
SELECT h.severity, COUNT(*)
FROM ops.tickets_header h
WHERE ` + where + `
GROUP BY h.severity
`
	rows2, err := r.db.QueryContext(ctx, qSev, args...)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var k string
		var v int64
		if err := rows2.Scan(&k, &v); err != nil {
			return nil, err
		}
		bySev[k] = v
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	return &models.TicketsSummaryResponse{
		Total:      total,
		ByStatus:   byStatus,
		BySeverity: bySev,
	}, nil
}