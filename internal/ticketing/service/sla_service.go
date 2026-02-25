package service

import (
	"context"
	"database/sql"

	"github.com/picopubliccloud/alarm-api/internal/ticketing/repo"
)

type SLAService struct {
	db      *sql.DB
	SLARepo *repo.SLARepo
}

func NewSLAService(db *sql.DB) *SLAService {
	return &SLAService{
		db:      db,
		SLARepo: repo.NewSLARepo(db),
	}
}

func (s *SLAService) StartForTicket(ctx context.Context, ticketID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.SLARepo.Start(ctx, tx, ticketID); err != nil {
		return err
	}

	return tx.Commit()
}
