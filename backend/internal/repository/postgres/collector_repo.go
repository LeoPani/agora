package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/LeoPani/agora/backend/internal/domain"
)

type CollectorRepo struct {
	db *sql.DB
}

func NewCollectorRepo(db *sql.DB) *CollectorRepo {
	return &CollectorRepo{db: db}
}

func (r *CollectorRepo) StartRun(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO collector_runs (collector_name, started_at, status)
		 VALUES ($1, NOW(), 'running') RETURNING id`,
		name,
	).Scan(&id)
	return id, err
}

func (r *CollectorRepo) FinishRun(ctx context.Context, run *domain.CollectorRun) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE collector_runs
		 SET finished_at = $1, status = $2, records_collected = $3, error_message = $4
		 WHERE id = $5`,
		now, run.Status, run.RecordsCollected, nullStr(run.ErrorMessage), run.ID,
	)
	return err
}
