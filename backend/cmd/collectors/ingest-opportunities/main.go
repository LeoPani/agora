// agora-ingest-opportunities lê editais_*.jsonl e popula a tabela opportunities.
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
	"sort"
	"time"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	pg "github.com/LeoPani/agora/backend/internal/repository/postgres"
)

const dataDir = "../ai-service/data"

type rawOpportunity struct {
	Source      string          `json:"source"`
	ExternalID  string          `json:"external_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	URL         string          `json:"url"`
	Deadline    string          `json:"deadline"`
	Status      string          `json:"status"`
	RawData     json.RawMessage `json:"raw_data"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-opportunities: fatal", "err", err)
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
	runID, err := collectorRepo.StartRun(ctx, "ingest-opportunities")
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}

	total, ingestErr := ingest(ctx, log, db)

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
	log.Info("ingest-opportunities complete", "total", total)
	return nil
}

func ingest(ctx context.Context, log *slog.Logger, db *sql.DB) (int, error) {
	pattern := filepath.Join(dataDir, "editais_*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return 0, err
	}
	sort.Strings(files)

	if len(files) == 0 {
		log.Warn("nenhum arquivo editais_*.jsonl encontrado", "dir", dataDir)
		return 0, nil
	}
	log.Info("arquivos de editais encontrados", "count", len(files))

	total := 0
	for _, path := range files {
		n, err := ingestFile(ctx, log, db, path)
		if err != nil {
			log.Warn("ingest file", "path", path, "err", err)
		}
		total += n
	}
	return total, nil
}

func ingestFile(ctx context.Context, log *slog.Logger, db *sql.DB, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	inserted := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var op rawOpportunity
		if err := json.Unmarshal(line, &op); err != nil {
			continue
		}
		if op.Title == "" || op.Source == "" {
			continue
		}
		if op.ExternalID == "" {
			op.ExternalID = op.URL
		}
		if op.ExternalID == "" {
			op.ExternalID = op.Title[:min(80, len(op.Title))]
		}

		rawJSON := op.RawData
		if rawJSON == nil {
			rawJSON = json.RawMessage("{}")
		}

		_, err := db.ExecContext(ctx, `
			INSERT INTO opportunities
			  (source, external_id, title, description, url, status, raw_data)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (source, external_id) DO UPDATE SET
			  title       = EXCLUDED.title,
			  description = EXCLUDED.description,
			  url         = EXCLUDED.url,
			  status      = EXCLUDED.status`,
			op.Source,
			op.ExternalID,
			op.Title,
			nullStr(op.Description),
			nullStr(op.URL),
			statusOr(op.Status),
			string(rawJSON),
		)
		if err != nil {
			log.Warn("upsert opportunity", "title", op.Title[:min(40, len(op.Title))], "err", err)
			continue
		}
		inserted++
	}

	log.Info("file done", "path", filepath.Base(path), "inserted", inserted)
	return inserted, scanner.Err()
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func statusOr(s string) string {
	if s == "" {
		return "aberto"
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
