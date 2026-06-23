// agora-ingest-trends lê market_trends.jsonl e popula a tabela market_trends.
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

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	pg "github.com/LeoPani/agora/backend/internal/repository/postgres"
)

const dataDir = "../ai-service/data"

type rawTrend struct {
	Keyword        string          `json:"keyword"`
	Geo            string          `json:"geo"`
	Timeframe      string          `json:"timeframe"`
	AvgInterest    int             `json:"avg_interest"`
	PeakInterest   int             `json:"peak_interest"`
	GrowthPct      float64         `json:"growth_pct"`
	RelatedQueries json.RawMessage `json:"related_queries"`
	RelatedTopics  json.RawMessage `json:"related_topics"`
	UFVDepartment  string          `json:"ufv_department"`
	RawData        json.RawMessage `json:"raw_data"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-trends: fatal", "err", err)
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
	runID, err := collectorRepo.StartRun(ctx, "ingest-trends")
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
	log.Info("ingest-trends complete", "total", total)
	return nil
}

func ingest(ctx context.Context, log *slog.Logger, db *sql.DB) (int, error) {
	path := filepath.Join(dataDir, "market_trends.jsonl")
	trends, err := loadJSONL(path)
	if err != nil {
		return 0, err
	}
	log.Info("loaded trends", "count", len(trends))

	upserted := 0
	for _, t := range trends {
		if t.Keyword == "" {
			continue
		}

		geo       := t.Geo
		timeframe := t.Timeframe
		if geo == "" {
			geo = "BR"
		}
		if timeframe == "" {
			timeframe = "today 5-y"
		}

		rqJSON := t.RelatedQueries
		if rqJSON == nil {
			rqJSON = json.RawMessage("[]")
		}
		rtJSON := t.RelatedTopics
		if rtJSON == nil {
			rtJSON = json.RawMessage("[]")
		}
		rawJSON := t.RawData
		if rawJSON == nil {
			rawJSON = json.RawMessage("{}")
		}

		_, err := db.ExecContext(ctx, `
			INSERT INTO market_trends
			  (keyword, geo, timeframe, avg_interest, peak_interest, growth_pct,
			   related_queries, related_topics, ufv_department, raw_data)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (keyword, geo, timeframe) DO UPDATE SET
			  avg_interest    = EXCLUDED.avg_interest,
			  peak_interest   = EXCLUDED.peak_interest,
			  growth_pct      = EXCLUDED.growth_pct,
			  related_queries = EXCLUDED.related_queries,
			  related_topics  = EXCLUDED.related_topics,
			  collected_at    = NOW()`,
			t.Keyword,
			geo,
			timeframe,
			t.AvgInterest,
			t.PeakInterest,
			t.GrowthPct,
			string(rqJSON),
			string(rtJSON),
			nullStr(t.UFVDepartment),
			string(rawJSON),
		)
		if err != nil {
			log.Warn("upsert trend", "keyword", t.Keyword, "err", err)
			continue
		}
		upserted++
	}

	log.Info("ingest-trends done", "upserted", upserted)
	return upserted, nil
}

func loadJSONL(path string) ([]rawTrend, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var out []rawTrend
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var t rawTrend
		if err := json.Unmarshal(line, &t); err != nil {
			continue
		}
		out = append(out, t)
	}
	return out, scanner.Err()
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
