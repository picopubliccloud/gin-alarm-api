package jobs

import (
	"context"
	"log"
	"time"

	"github.com/picopubliccloud/alarm-api/internal/ticketing/repo"
)

type OutboxWorker struct {
	Repo      *repo.OutboxRepo
	BatchSize int
	Interval  time.Duration
}

func NewOutboxWorker(r *repo.OutboxRepo) *OutboxWorker {
	return &OutboxWorker{
		Repo:      r,
		BatchSize: 100,
		Interval:  2 * time.Second,
	}
}

func (w *OutboxWorker) Run(ctx context.Context) {
	t := time.NewTicker(w.Interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n, err := w.Repo.Process(ctx, w.BatchSize)
			if err != nil {
				log.Printf("outbox worker error: %v", err)
				continue
			}
			_ = n
		}
	}
}
