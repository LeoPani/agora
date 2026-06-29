// agora-ingest-partners lê partners_*.jsonl e popula a tabela partners.
package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	pg "github.com/LeoPani/agora/backend/internal/repository/postgres"
)

var dataDir = config.DataDir()

type rawPartner struct {
	Name           string          `json:"name"`
	NormalizedName string          `json:"normalized_name"`
	CNPJ           string          `json:"cnpj"`
	PartnerType    string          `json:"partner_type"`
	Sector         string          `json:"sector"`
	Location       string          `json:"location"`
	CNAECode       string          `json:"cnae_code"`
	LattesURL      string          `json:"lattes_url"`
	LattesID       string          `json:"lattes_id"`
	LinkedInURL    string          `json:"linkedin_url"`
	ContactEmail   string          `json:"contact_email"`
	UFVAreas       []string        `json:"ufv_areas"`
	InterestScore  float64         `json:"interest_score"`
	Source         string          `json:"source"`
	RawData        json.RawMessage `json:"raw_data"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-partners: fatal", "err", err)
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
	runID, err := collectorRepo.StartRun(ctx, "ingest-partners")
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
	log.Info("ingest-partners complete", "total", total)
	return nil
}

func ingest(ctx context.Context, log *slog.Logger, db *sql.DB) (int, error) {
	pattern := filepath.Join(dataDir, "partners_*.jsonl")
	files, _ := filepath.Glob(pattern)
	sort.Strings(files)

	if len(files) == 0 {
		log.Warn("nenhum arquivo partners_*.jsonl", "dir", dataDir)
		return 0, nil
	}

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
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var p rawPartner
		if err := json.Unmarshal(line, &p); err != nil {
			continue
		}
		if p.Name == "" {
			continue
		}
		if p.NormalizedName == "" {
			p.NormalizedName = strings.ToLower(strings.TrimSpace(p.Name))
		}

		rawJSON := p.RawData
		if rawJSON == nil {
			rawJSON = json.RawMessage("{}")
		}

		score := p.InterestScore
		if score == 0 {
			score = 0.3 + math.Min(float64(len(p.UFVAreas))*0.1, 0.5)
		}

		var partnerID int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO partners
			  (name, normalized_name, cnpj, partner_type, sector, location,
			   cnae_code, lattes_id, linkedin_url, contact_email,
			   interest_score, source, raw_data)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
			ON CONFLICT (normalized_name) DO UPDATE SET
			  sector         = EXCLUDED.sector,
			  location       = COALESCE(EXCLUDED.location, partners.location),
			  interest_score = GREATEST(EXCLUDED.interest_score, partners.interest_score),
			  source         = EXCLUDED.source
			RETURNING id`,
			p.Name,
			p.NormalizedName,
			nullStr(p.CNPJ),
			nullStr(p.PartnerType),
			nullStr(p.Sector),
			nullStr(p.Location),
			nullStr(p.CNAECode),
			nullStr(p.LattesID),
			nullStr(p.LinkedInURL),
			nullStr(p.ContactEmail),
			score,
			nullStr(p.Source),
			string(rawJSON),
		).Scan(&partnerID)
		if err != nil {
			log.Warn("upsert partner", "name", p.Name[:min(40, len(p.Name))], "err", err)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
