package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/lib/pq"
)

type PublicationRepo struct {
	db *sql.DB
}

func NewPublicationRepo(db *sql.DB) *PublicationRepo {
	return &PublicationRepo{db: db}
}

func (r *PublicationRepo) Upsert(ctx context.Context, pub *domain.Publication) (int64, error) {
	topicsJSON, err := json.Marshal(pub.Topics)
	if err != nil {
		topicsJSON = []byte("[]")
	}

	const q = `
		INSERT INTO publications (openalex_id, doi, title, abstract, publication_year, type, cited_by_count, topics)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (openalex_id) DO UPDATE SET
			cited_by_count = EXCLUDED.cited_by_count,
			topics         = EXCLUDED.topics
		RETURNING id`

	var id int64
	err = r.db.QueryRowContext(ctx, q,
		pub.OpenAlexID,
		nullStr(pub.DOI),
		pub.Title,
		nullStr(pub.Abstract),
		nullIntYear(pub.PublicationYear),
		nullStr(pub.Type),
		pub.CitedByCount,
		topicsJSON,
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

// UpsertWithSource upsert por openalex_id e grava o campo source.
func (r *PublicationRepo) UpsertWithSource(ctx context.Context, pub *domain.Publication, source string) (int64, error) {
	topicsJSON, err := json.Marshal(pub.Topics)
	if err != nil {
		topicsJSON = []byte("[]")
	}

	const q = `
		INSERT INTO publications (openalex_id, doi, title, abstract, publication_year, type, cited_by_count, topics, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (openalex_id) DO UPDATE SET
			cited_by_count = EXCLUDED.cited_by_count,
			topics         = EXCLUDED.topics,
			source         = EXCLUDED.source
		RETURNING id`

	var id int64
	err = r.db.QueryRowContext(ctx, q,
		pub.OpenAlexID,
		nullStr(pub.DOI),
		pub.Title,
		nullStr(pub.Abstract),
		nullIntYear(pub.PublicationYear),
		nullStr(pub.Type),
		pub.CitedByCount,
		topicsJSON,
		source,
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

func (r *PublicationRepo) LinkAuthor(ctx context.Context, pa *domain.PublicationAuthor) error {
	const q = `
		INSERT INTO publication_authors (publication_id, researcher_id, author_position, is_corresponding)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (publication_id, researcher_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, q, pa.PublicationID, pa.ResearcherID, pa.AuthorPosition, pa.IsCorresponding)
	return err
}

func nullIntYear(y int) interface{} {
	if y == 0 {
		return nil
	}
	return y
}
