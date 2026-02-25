package repo

import (
	"context"
	"database/sql"
)

type TxBeginner interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
}

func beginTx(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	return db.BeginTx(ctx, nil)
}
