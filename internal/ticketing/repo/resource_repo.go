package repo

import "database/sql"

type ResourceRepo struct{ db *sql.DB }

func NewResourceRepo(db *sql.DB) *ResourceRepo {
	return &ResourceRepo{db: db}
}
