package repo

import (
	"context"
	"database/sql"

	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
)

type UserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

// Upsert user (critical flow): external_subject = Keycloak sub
func (r *UserRepo) Upsert(ctx context.Context, sub, email, displayName string, actor models.ActorType) (*models.User, error) {
	const q = `
INSERT INTO ops.users (external_subject, email, display_name, actor_type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (external_subject)
DO UPDATE SET
  email = EXCLUDED.email,
  display_name = EXCLUDED.display_name,
  actor_type = EXCLUDED.actor_type,
  is_active = true
RETURNING user_id, external_subject, COALESCE(email::text,''), COALESCE(display_name,''), actor_type::text, is_active;
`
	u := &models.User{}
	var actorType string
	if err := r.db.QueryRowContext(ctx, q, sub, nullIfEmpty(email), nullIfEmpty(displayName), string(actor)).
		Scan(&u.UserID, &u.ExternalSubject, &u.Email, &u.DisplayName, &actorType, &u.IsActive); err != nil {
		return nil, err
	}
	u.ActorType = models.ActorType(actorType)
	return u, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
