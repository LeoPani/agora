// agora-ingest-openalex lê os JSONL gerados pelo coletor Python
// (ai-service/data/openalex_*.jsonl) e upserta no Postgres.
//
// Usage:
//
//	go run ./cmd/collectors/ingest-openalex
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	pg "github.com/LeoPani/agora/backend/internal/repository/postgres"
)

var dataDir = config.DataDir()

// jsonl shapes from openalex_collector.py

type rawResearcher struct {
	OpenAlexID     string `json:"openalex_id"`
	ORCID          string `json:"orcid"`
	FullName       string `json:"full_name"`
	NormalizedName string `json:"normalized_name"`
	Department     string `json:"department"`
}

type rawTopic struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

type rawPublication struct {
	OpenAlexID      string     `json:"openalex_id"`
	DOI             string     `json:"doi"`
	Title           string     `json:"title"`
	Abstract        string     `json:"abstract"`
	PublicationYear int        `json:"publication_year"`
	Type            string     `json:"type"`
	CitedByCount    int        `json:"cited_by_count"`
	Topics          []rawTopic `json:"topics"`
}

type rawCoauthorship struct {
	PublicationOpenAlexID      string `json:"publication_openalex_id"`
	ResearcherNormalizedName   string `json:"researcher_normalized_name"`
	Position                   int    `json:"position"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-openalex: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		DSN:             cfg.DatabaseURL,
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		PingTimeout:     5 * time.Second,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	collectorRepo := pg.NewCollectorRepo(db)
	researcherRepo := pg.NewResearcherRepo(db)
	publicationRepo := pg.NewPublicationRepo(db)

	runID, err := collectorRepo.StartRun(ctx, "ingest-openalex")
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}

	total, ingestErr := ingest(ctx, log, researcherRepo, publicationRepo)

	status := "ok"
	errMsg := ""
	if ingestErr != nil {
		status = "error"
		errMsg = ingestErr.Error()
	}

	finishErr := collectorRepo.FinishRun(ctx, &domain.CollectorRun{
		ID:               runID,
		Status:           status,
		RecordsCollected: total,
		ErrorMessage:     errMsg,
	})
	if finishErr != nil {
		log.Warn("finish run failed", "err", finishErr)
	}

	if ingestErr != nil {
		return ingestErr
	}

	log.Info("ingest complete", "total_records", total)
	return nil
}

func ingest(
	ctx context.Context,
	log *slog.Logger,
	resRepo *pg.ResearcherRepo,
	pubRepo *pg.PublicationRepo,
) (int, error) {
	// 1. Load researchers
	researchers, err := loadJSONL[rawResearcher](filepath.Join(dataDir, "openalex_researchers.jsonl"))
	if err != nil {
		return 0, fmt.Errorf("load researchers: %w", err)
	}

	// ID map: normalized_name → DB id
	resIDMap := make(map[string]int64, len(researchers))
	resInserted, resDupes := 0, 0

	for _, r := range researchers {
		if r.NormalizedName == "" {
			continue
		}
		id, err := resRepo.Upsert(ctx, &domain.Researcher{
			OpenAlexID:     r.OpenAlexID,
			ORCID:          r.ORCID,
			FullName:       r.FullName,
			NormalizedName: r.NormalizedName,
			Department:     r.Department,
			Institution:    "UFV",
		})
		if err != nil {
			if errors.Is(err, domain.ErrDuplicate) {
				resDupes++
				// Still need the id for linking
				id2, _ := resRepo.FindByNormalizedName(ctx, r.NormalizedName)
				resIDMap[r.NormalizedName] = id2
				continue
			}
			log.Warn("upsert researcher", "name", r.NormalizedName, "err", err)
			continue
		}
		resIDMap[r.NormalizedName] = id
		resInserted++
	}
	log.Info("researchers done", "inserted", resInserted, "dupes", resDupes)

	// 2. Load publications
	publications, err := loadJSONL[rawPublication](filepath.Join(dataDir, "openalex_publications.jsonl"))
	if err != nil {
		return 0, fmt.Errorf("load publications: %w", err)
	}

	// ID map: openalex_id → DB id
	pubIDMap := make(map[string]int64, len(publications))
	pubInserted, pubDupes := 0, 0

	for _, p := range publications {
		if p.OpenAlexID == "" || p.Title == "" {
			continue
		}
		topicsJSON, _ := json.Marshal(p.Topics)
		id, err := pubRepo.Upsert(ctx, &domain.Publication{
			OpenAlexID:      p.OpenAlexID,
			DOI:             p.DOI,
			Title:           p.Title,
			Abstract:        p.Abstract,
			PublicationYear: p.PublicationYear,
			Type:            p.Type,
			CitedByCount:    p.CitedByCount,
			Topics:          topicsJSON,
		})
		if err != nil {
			if errors.Is(err, domain.ErrDuplicate) {
				pubDupes++
				continue
			}
			log.Warn("upsert publication", "id", p.OpenAlexID, "err", err)
			continue
		}
		pubIDMap[p.OpenAlexID] = id
		pubInserted++
	}
	log.Info("publications done", "inserted", pubInserted, "dupes", pubDupes)

	// 3. Load coauthorships
	coauthorships, err := loadJSONL[rawCoauthorship](filepath.Join(dataDir, "openalex_coauthorships.jsonl"))
	if err != nil {
		return 0, fmt.Errorf("load coauthorships: %w", err)
	}

	linked := 0
	for _, ca := range coauthorships {
		pubID, okP := pubIDMap[ca.PublicationOpenAlexID]
		resID, okR := resIDMap[ca.ResearcherNormalizedName]
		if !okP || !okR {
			continue
		}
		if err := pubRepo.LinkAuthor(ctx, &domain.PublicationAuthor{
			PublicationID:  pubID,
			ResearcherID:   resID,
			AuthorPosition: ca.Position,
		}); err != nil {
			log.Warn("link author", "err", err)
			continue
		}
		linked++
	}
	log.Info("authors linked", "count", linked)

	return pubInserted + resInserted + linked, nil
}

func loadJSONL[T any](path string) ([]T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var out []T
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var item T
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, fmt.Errorf("unmarshal line: %w", err)
		}
		out = append(out, item)
	}
	return out, scanner.Err()
}
