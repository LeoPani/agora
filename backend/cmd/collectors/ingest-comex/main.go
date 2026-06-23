// agora-ingest-comex lê comex_import_gaps.jsonl e popula a tabela import_gaps.
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
	"time"

	"github.com/lib/pq"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	pg "github.com/LeoPani/agora/backend/internal/repository/postgres"
)

const dataDir = "../ai-service/data"

type rawGap struct {
	SH4Code          string          `json:"sh4_code"`
	Description      string          `json:"description"`
	CountryOrigin    string          `json:"country_origin"`
	Year             int             `json:"year"`
	ImportValueUSD   float64         `json:"import_value_usd"`
	ImportKG         float64         `json:"import_kg"`
	UFVRelatedAreas  []string        `json:"ufv_related_areas"`
	OpportunityScore float64         `json:"opportunity_score"`
	RawData          json.RawMessage `json:"raw_data"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-comex: fatal", "err", err)
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
	runID, err := collectorRepo.StartRun(ctx, "ingest-comex")
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
	log.Info("ingest-comex complete", "total", total)
	return nil
}

func ingest(ctx context.Context, log *slog.Logger, db *sql.DB) (int, error) {
	path := filepath.Join(dataDir, "comex_import_gaps.jsonl")
	gaps, err := loadJSONL(path)
	if err != nil {
		return 0, err
	}
	log.Info("loaded import gaps", "count", len(gaps))

	inserted := 0
	for _, g := range gaps {
		if g.SH4Code == "" || g.Year == 0 {
			continue
		}

		rawJSON := g.RawData
		if rawJSON == nil {
			rawJSON = json.RawMessage("{}")
		}

		_, err := db.ExecContext(ctx, `
			INSERT INTO import_gaps
			  (sh4_code, description, country_origin, year,
			   import_value_usd, import_kg, ufv_related_areas, opportunity_score, raw_data)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (sh4_code, country_origin, year) DO UPDATE SET
			  import_value_usd   = EXCLUDED.import_value_usd,
			  import_kg          = EXCLUDED.import_kg,
			  opportunity_score  = EXCLUDED.opportunity_score,
			  ufv_related_areas  = EXCLUDED.ufv_related_areas`,
			g.SH4Code,
			nullStr(g.Description),
			g.CountryOrigin,
			g.Year,
			g.ImportValueUSD,
			g.ImportKG,
			pq.Array(g.UFVRelatedAreas),
			g.OpportunityScore,
			string(rawJSON),
		)
		if err != nil {
			log.Warn("upsert import_gap", "sh4", g.SH4Code, "err", err)
			continue
		}
		inserted++
	}

	log.Info("ingest-comex done", "inserted", inserted)
	return inserted, nil
}

func loadJSONL(path string) ([]rawGap, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var out []rawGap
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var g rawGap
		if err := json.Unmarshal(line, &g); err != nil {
			continue
		}
		out = append(out, g)
	}
	return out, scanner.Err()
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
