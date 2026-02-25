// /internal/ticketing/service/ticket_service.go
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/repo"
)

func strPtr(s string) *string { return &s }

type TicketService struct {
	db *sql.DB

	TicketRepo     *repo.TicketRepo
	UpdateRepo     *repo.UpdateRepo
	AssignmentRepo *repo.AssignmentRepo
	LockRepo       *repo.LockRepo
	ClosureRepo    *repo.ClosureRepo
	AttachmentRepo *repo.AttachmentRepo
	OutboxRepo     *repo.OutboxRepo
	RBACRepo       *repo.RBACRepo
	meta           *repo.MetaRepo
}

func NewTicketService(
	db *sql.DB,
	ticketRepo *repo.TicketRepo,
	updateRepo *repo.UpdateRepo,
	assignRepo *repo.AssignmentRepo,
	lockRepo *repo.LockRepo,
	closureRepo *repo.ClosureRepo,
	attachRepo *repo.AttachmentRepo,
	outboxRepo *repo.OutboxRepo,
	rbacRepo *repo.RBACRepo,
	metaRepo *repo.MetaRepo,
) *TicketService {
	return &TicketService{
		db:             db,
		TicketRepo:     ticketRepo,
		UpdateRepo:     updateRepo,
		AssignmentRepo: assignRepo,
		LockRepo:       lockRepo,
		ClosureRepo:    closureRepo,
		AttachmentRepo: attachRepo,
		OutboxRepo:     outboxRepo,
		RBACRepo:       rbacRepo,
		meta:           metaRepo,
	}
}

//
// LISTING
//

func (s *TicketService) ListTickets(ctx context.Context, filter repo.TicketListFilter) ([]models.TicketListItem, string, bool, error) {
	return s.TicketRepo.List(ctx, filter)
}

func (s *TicketService) ListPoolTickets(ctx context.Context, filter repo.TicketListFilter) ([]models.TicketListItem, string, bool, error) {
	filter.OnlyPool = true
	return s.TicketRepo.List(ctx, filter)
}

//
// READ
//

func (s *TicketService) GetTicket(ctx context.Context, ticketID string, allowedProjects []string, maxVis models.TicketVisibility) (*models.TicketDetail, error) {
	if ticketID == "" {
		return nil, errors.New("ticket_id required")
	}
	return s.TicketRepo.Get(ctx, ticketID, allowedProjects, maxVis)
}

//
// CREATE
//

func (s *TicketService) CreateTicket(ctx context.Context, req *models.CreateTicketRequest, userID string, actor models.ActorType) (string, int64, error) {
	if req == nil {
		return "", 0, errors.New("request is nil")
	}
	if userID == "" {
		return "", 0, errors.New("user_id required")
	}

	if strings.TrimSpace(req.ProjectID) == "" {
		cid, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
		if err != nil {
			return "", 0, fmt.Errorf("invalid customer_id")
		}

		pid, err := s.meta.GetDefaultProjectIDByCustomer(ctx, cid)
		if err != nil {
			return "", 0, fmt.Errorf("failed to resolve project_id for customer")
		}

		req.ProjectID = pid
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = tx.Rollback() }()

	header, text := req.ToModels(userID, actor)

	ticketID, ticketNumber, err := s.TicketRepo.Create(ctx, tx, header, text)
	if err != nil {
		return "", 0, err
	}

	_ = s.OutboxRepo.Emit(ctx, tx, "TICKET", ticketID, "TICKET_CREATED", map[string]any{
		"ticket_id":     ticketID,
		"ticket_number": ticketNumber,
	})

	if err := tx.Commit(); err != nil {
		return "", 0, err
	}
	return ticketID, ticketNumber, nil
}

//
// UPDATES
//

func (s *TicketService) AddUpdate(ctx context.Context, ticketID string, req *models.AddUpdateRequest, userID string, actor models.ActorType) error {
	if ticketID == "" {
		return errors.New("ticket_id required")
	}
	if req == nil {
		return errors.New("request is nil")
	}
	if userID == "" {
		return errors.New("user_id required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// If this is a STATUS update → update tickets_header.status
	if strings.EqualFold(string(req.UpdateType), "STATUS_CHANGE") {
		newStatus, err := extractStatusFromUpdate(req)
		if err != nil {
			return err
		}

		if err := s.TicketRepo.UpdateStatus(ctx, tx, ticketID, models.TicketStatus(newStatus), userID); err != nil {
			return err
		}

		if req.Structured == nil {
			req.Structured = map[string]any{}
		}
		if m, ok := req.Structured.(map[string]any); ok {
			m["status"] = newStatus
		}
	}

	// Insert update row (audit log)
	if err := s.UpdateRepo.Insert(ctx, tx, ticketID, req, userID, actor); err != nil {
		return err
	}

	_ = s.OutboxRepo.Emit(ctx, tx, "TICKET", ticketID, "TICKET_UPDATED", map[string]any{
		"ticket_id": ticketID,
		"by":        userID,
	})

	return tx.Commit()
}

func extractStatusFromUpdate(req *models.AddUpdateRequest) (string, error) {
	// Prefer structured.status
	if req.Structured != nil {
		if m, ok := req.Structured.(map[string]any); ok {
			if v, ok := m["status"]; ok {
				st := strings.TrimSpace(fmt.Sprintf("%v", v))
				if st != "" {
					return strings.ToUpper(st), nil
				}
			}
		}
	}

	// fallback: body contains status
	if req.Body != nil {
		st := strings.TrimSpace(*req.Body)
		if st != "" {
			return strings.ToUpper(st), nil
		}
	}

	return "", errors.New("STATUS update requires structured.status or body")
}

//
// ASSIGNMENT
//

func (s *TicketService) AssignOwner(ctx context.Context, ticketID string, targetUserID string, actorUserID string) error {
	if ticketID == "" || targetUserID == "" || actorUserID == "" {
		return errors.New("ticket_id, target_user_id, actor_user_id required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Resolve display name for better history message
	targetName := targetUserID
	if s.AssignmentRepo != nil {
		if dn, err := s.AssignmentRepo.GetUserDisplayName(ctx, tx, targetUserID); err == nil && strings.TrimSpace(dn) != "" {
			targetName = dn
		}
	}

	// Assign/reassign in DB (repo should end old OWNER if needed)
	if err := s.AssignmentRepo.AssignOwner(ctx, tx, ticketID, targetUserID, actorUserID); err != nil {
		return err
	}

	// Add update history row (same TX)
	up := &models.AddUpdateRequest{
		UpdateType: models.UpdateType("FIELD_CHANGE"),
		Visibility: models.VisibilityInternal,
		Body:       strPtr(fmt.Sprintf("Owner assigned to %s", targetName)),
		Structured: map[string]any{
			"event":          "OWNER_CHANGE",
			"action":         "ASSIGN",
			"target_user_id": targetUserID,
			"target_name":    targetName, // optional (handy for UI)
		},
	}

	// IMPORTANT: created_by_actor should be the ACTOR, not "SYSTEM"
	// so it shows who did it (NETOPS/...) not SYSTEM.
	if err := s.UpdateRepo.Insert(ctx, tx, ticketID, up, actorUserID, models.ActorType("NETOPS")); err != nil {
		// If you already have actor type available from middleware, pass it here instead of "NETOPS".
		// For now keeping NETOPS as you showed in logs. You can replace with a parameter later.
		return err
	}

	_ = s.OutboxRepo.Emit(ctx, tx, "TICKET", ticketID, "TICKET_ASSIGNED", map[string]any{
		"ticket_id": ticketID,
		"user_id":   targetUserID,
		"by":        actorUserID,
	})

	_ = s.TicketRepo.UpdateStatus(ctx, tx, ticketID, models.TicketStatus("ACKNOWLEDGED"), actorUserID)

	return tx.Commit()
}

func (s *TicketService) UnassignOwner(
	ctx context.Context,
	ticketID string,
	unassignReasonID *int16,
	note *string,
	actorUserID string,
) error {

	if ticketID == "" || actorUserID == "" {
		return errors.New("ticket_id and actor_user_id required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.AssignmentRepo.Unassign(ctx, tx, ticketID, actorUserID, unassignReasonID, note); err != nil {
		return err
	}

	// Add update history row (same TX)
	msg := "Owner unassigned"
	reasonName := "" //  declare here so it can be used in Structured too

	if unassignReasonID != nil {
		if s.AssignmentRepo != nil {
			if rn, err := s.AssignmentRepo.GetUnassignReasonName(ctx, tx, *unassignReasonID); err == nil && strings.TrimSpace(rn) != "" {
				reasonName = rn
			}
		}

		if reasonName != "" {
			msg = fmt.Sprintf("Owner unassigned (%s)", reasonName)
		} else {
			// fallback if lookup fails
			msg = fmt.Sprintf("Owner unassigned (reason_id=%d)", *unassignReasonID)
		}
	}

	if note != nil && strings.TrimSpace(*note) != "" {
		msg = msg + ": " + strings.TrimSpace(*note)
	}

	up := &models.AddUpdateRequest{
		UpdateType: models.UpdateType("FIELD_CHANGE"),
		Visibility: models.VisibilityInternal,
		Body:       strPtr(msg),
		Structured: map[string]any{
			"event":              "OWNER_CHANGE",
			"action":             "UNASSIGN",
			"unassign_reason_id": unassignReasonID,
			"unassign_reason":    reasonName, // ✅ now valid
			"note":               note,
		},
	}

	// IMPORTANT: created_by_actor should be the ACTOR, not "SYSTEM"
	if err := s.UpdateRepo.Insert(ctx, tx, ticketID, up, actorUserID, models.ActorType("NETOPS")); err != nil {
		// replace NETOPS with your real actor type if you have it available here.
		return err
	}

	_ = s.OutboxRepo.Emit(ctx, tx, "TICKET", ticketID, "TICKET_UNASSIGNED", map[string]any{
		"ticket_id":          ticketID,
		"by":                 actorUserID,
		"unassign_reason_id": unassignReasonID,
		"note":               note,
	})

	return tx.Commit()
}

//
// LOCKS
//

func (s *TicketService) LockTicket(ctx context.Context, ticketID string, userID string, LockReasonID int16, ttl int64) error {
	if ticketID == "" || userID == "" {
		return errors.New("ticket_id and user_id required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.LockRepo.TryLock(ctx, tx, ticketID, userID, LockReasonID, ttl); err != nil {
		return err
	}

	_ = s.TicketRepo.UpdateStatus(ctx, tx, ticketID, models.TicketStatus("IN_PROGRESS"), userID)

	return tx.Commit()
}

func (s *TicketService) UnlockTicket(ctx context.Context, ticketID string, userID string) error {
	if ticketID == "" || userID == "" {
		return errors.New("ticket_id and user_id required")
	}
	return s.LockRepo.Release(ctx, ticketID, userID)
}

//
// CLOSURE
//

func (s *TicketService) CloseTicket(
	ctx context.Context,
	ticketID string,
	req *models.CloseTicketRequest,
	userID string,
) error {

	if ticketID == "" || userID == "" {
		return errors.New("ticket_id and user_id required")
	}
	if req == nil {
		return errors.New("request is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 1) closure summary FIRST (DB trigger dependency)
	if err := s.ClosureRepo.Upsert(ctx, tx, ticketID, req, userID); err != nil {
		return err
	}

	// 2) optional verification attachment
	if req.VerificationAttachment != nil && s.AttachmentRepo != nil {
		if err := s.AttachmentRepo.InsertVerification(ctx, tx, ticketID, userID, req.VerificationAttachment); err != nil {
			return err
		}
	}

	// 3) close ticket header
	closedAt := time.Now().UTC()
	if err := s.TicketRepo.Close(ctx, tx, ticketID, req.ResolutionCode, userID, closedAt); err != nil {
		return err
	}

	// 4) outbox
	_ = s.OutboxRepo.Emit(ctx, tx,
		"TICKET",
		ticketID,
		"TICKET_CLOSED",
		map[string]any{
			"ticket_id": ticketID,
			"closed_by": userID,
		},
	)

	return tx.Commit()
}


func (s *TicketService) SummaryTickets(
	ctx context.Context,
	pool bool,
	status string,
	severity string,
	includeClosed bool,
	allowedProjects []string,
	maxVis models.TicketVisibility,
) (*models.TicketsSummaryResponse, error) {

	// normalize
	status = strings.ToUpper(strings.TrimSpace(status))
	severity = strings.ToUpper(strings.TrimSpace(severity))

	return s.TicketRepo.Summary(ctx, repo.TicketSummaryFilter{
		OnlyPool:      pool,
		Status:        status,
		Severity:      severity,
		IncludeClosed: includeClosed,
		AllowedProjects: allowedProjects,
		MaxVis:        maxVis,
	})
}