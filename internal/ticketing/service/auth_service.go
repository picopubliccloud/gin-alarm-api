package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/repo"
)

type RBACContext struct {
	UserID             string
	ExternalSubject    string
	Actor              models.ActorType
	AllowedProjectIDs  []string // nil/empty => unrestricted (OPS)
	MaxVisibility      models.TicketVisibility
	Capabilities       repo.UserCapabilities
}

type AuthService struct {
	db       *sql.DB
	userRepo *repo.UserRepo
	rbacRepo *repo.RBACRepo
}

func NewAuthService(db *sql.DB, userRepo *repo.UserRepo, rbacRepo *repo.RBACRepo) *AuthService {
	return &AuthService{db: db, userRepo: userRepo, rbacRepo: rbacRepo}
}

// IsOps helper
func (s *AuthService) IsOps(actor models.ActorType) bool {
	return strings.EqualFold(string(actor), "OPS")
}

// UpsertUser ensures Keycloak sub exists in ops.users.
// Returns internal user_id (uuid as string).
func (s *AuthService) UpsertUser(ctx context.Context, externalSubject string, email *string, displayName *string, actor models.ActorType) (string, error) {
	sub := strings.TrimSpace(externalSubject)
	if sub == "" {
		return "", errors.New("external_subject is required")
	}

	em := ""
	if email != nil {
		em = strings.TrimSpace(*email)
	}
	dn := ""
	if displayName != nil {
		dn = strings.TrimSpace(*displayName)
	}

	u, err := s.userRepo.Upsert(ctx, sub, em, dn, actor)
	if err != nil {
		return "", err
	}
	return u.UserID, nil
}

// ResolveRBAC: loads user_id + actor_type, then returns project scope + max visibility + capabilities.
func (s *AuthService) ResolveRBAC(ctx context.Context, externalSubject string) (*RBACContext, error) {
	sub := strings.TrimSpace(externalSubject)
	if sub == "" {
		return nil, errors.New("external_subject is required")
	}

	userID, actorStr, err := s.rbacRepo.GetUserActorType(ctx, sub)
	if err != nil {
		return nil, err
	}

	caps, _ := s.rbacRepo.GetUserCapabilities(ctx, userID) // if no memberships -> caps false

	out := &RBACContext{
		UserID:           userID,
		ExternalSubject:  sub,
		Actor:            models.ActorType(actorStr),
		Capabilities:     caps,
	}

	// CUSTOMER: restricted to memberships, PUBLIC visibility
	if strings.EqualFold(actorStr, "CUSTOMER") {
		pids, err := s.rbacRepo.GetUserProjectScope(ctx, userID)
		if err != nil {
			return nil, err
		}
		out.AllowedProjectIDs = pids
		out.MaxVisibility = models.VisibilityPublic
		return out, nil
	}

	// OPS/SYSTEM: unrestricted projects (nil/empty => unrestricted)
	out.AllowedProjectIDs = nil

	if caps.CanViewRestricted {
		out.MaxVisibility = models.VisibilityRestricted
	} else {
		out.MaxVisibility = models.VisibilityInternal
	}

	return out, nil
}
