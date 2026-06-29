// agora-enrich-patents lê google_patents_enriched.jsonl e atualiza a tabela patents
// com claims, description e cria patent_citations onde há citações a publicações.
package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	pg "github.com/LeoPani/agora/backend/internal/repository/postgres"
)

var dataDir = config.DataDir()

type enrichedPatent struct {
	INPINumber    string   `json:"inpi_number"`
	Found         bool     `json:"found"`
	Claims        string   `json:"claims"`
	Description   string   `json:"description"`
	FamilyMembers []string `json:"family_members"`
	CitedPatents  []string `json:"cited_patents"`
	CitedPapers   []string `json:"cited_papers"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("enrich-patents: fatal", "err", err)
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
	runID, err := collectorRepo.StartRun(ctx, "enrich-patents")
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}

	total, ingestErr := enrich(ctx, log, db)

	status := "ok"
	errMsg := ""
	if ingestErr != nil {
		status = "error"
		errMsg = ingestErr.Error()
	}
	_ = collectorRepo.FinishRun(ctx, &domain.CollectorRun{
		ID: runID, Status: status, RecordsCollected: total, ErrorMessage: errMsg,
	})
	if ingestErr != nil {
		return ingestErr
	}
	log.Info("enrich-patents complete", "updated", total)
	return nil
}

func enrich(ctx context.Context, log *slog.Logger, db *sql.DB) (int, error) {
	path := filepath.Join(dataDir, "google_patents_enriched.jsonl")
	records, err := loadJSONL(path)
	if err != nil {
		return 0, err
	}
	log.Info("loaded enriched patents", "count", len(records))

	updated := 0
	for _, rec := range records {
		if !rec.Found || rec.INPINumber == "" {
			continue
		}

		familyJSON, _ := json.Marshal(rec.FamilyMembers)
		citedPatentsJSON, _ := json.Marshal(rec.CitedPatents)

		_, err := db.ExecContext(ctx, `
			UPDATE patents
			SET claims      = $1,
			    description = $2,
			    raw_data    = COALESCE(raw_data, '{}') ||
			                  jsonb_build_object(
			                    'family_members',  $3::jsonb,
			                    'cited_patents',   $4::jsonb
			                  )
			WHERE inpi_number = $5`,
			nullStr(rec.Claims),
			nullStr(rec.Description),
			string(familyJSON),
			string(citedPatentsJSON),
			rec.INPINumber,
		)
		if err != nil {
			log.Warn("update patent", "inpi", rec.INPINumber, "err", err)
			continue
		}

		// Tentar linkar citações NPL a publicações existentes
		patentID, err := findPatentID(ctx, db, rec.INPINumber)
		if err != nil || patentID == 0 {
			updated++
			continue
		}

		for _, paper := range rec.CitedPapers {
			if paper == "" {
				continue
			}
			pubID, err := findPublicationByTitle(ctx, db, paper)
			if err != nil || pubID == 0 {
				continue
			}
			_, _ = db.ExecContext(ctx, `
				INSERT INTO patent_citations (patent_id, publication_id, source)
				VALUES ($1, $2, 'google_patents')
				ON CONFLICT DO NOTHING`, patentID, pubID)
		}

		updated++
	}

	log.Info("enrichment done", "updated", updated)
	return updated, nil
}

func findPatentID(ctx context.Context, db *sql.DB, inpiNumber string) (int64, error) {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT id FROM patents WHERE inpi_number = $1", inpiNumber).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func findPublicationByTitle(ctx context.Context, db *sql.DB, title string) (int64, error) {
	// Busca aproximada: título começa com os primeiros 50 chars do paper citado
	prefix := title
	if len(prefix) > 60 {
		prefix = prefix[:60]
	}
	// Simplificação: só tenta match exato na primeira palavra significativa
	words := strings.Fields(prefix)
	if len(words) < 3 {
		return 0, nil
	}
	query := strings.Join(words[:3], " ")

	var id int64
	err := db.QueryRowContext(ctx,
		"SELECT id FROM publications WHERE title ILIKE $1 LIMIT 1",
		"%"+query+"%",
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func loadJSONL(path string) ([]enrichedPatent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var out []enrichedPatent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var p enrichedPatent
		if err := json.Unmarshal(line, &p); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, scanner.Err()
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
