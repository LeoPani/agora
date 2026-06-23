package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/lib/pq"
)

type ResearcherRepo struct {
	db *sql.DB
}

func NewResearcherRepo(db *sql.DB) *ResearcherRepo {
	return &ResearcherRepo{db: db}
}

func (r *ResearcherRepo) Upsert(ctx context.Context, res *domain.Researcher) (int64, error) {
	const q = `
		INSERT INTO researchers (openalex_id, orcid, full_name, normalized_name, department, institution)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (normalized_name) DO UPDATE SET
			openalex_id = EXCLUDED.openalex_id,
			orcid       = EXCLUDED.orcid,
			full_name   = EXCLUDED.full_name,
			department  = COALESCE(EXCLUDED.department, researchers.department),
			updated_at  = NOW()
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		nullStr(res.OpenAlexID),
		nullStr(res.ORCID),
		res.FullName,
		res.NormalizedName,
		nullStr(res.Department),
		stringOr(res.Institution, "UFV"),
	).Scan(&id)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, domain.ErrDuplicate
		}
		return 0, err
	}
	return id, nil
}

func (r *ResearcherRepo) FindByNormalizedName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		"SELECT id FROM researchers WHERE normalized_name = $1", name).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, domain.ErrNotFound
	}
	return id, err
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func stringOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
