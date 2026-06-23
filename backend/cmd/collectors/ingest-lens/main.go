// agora-ingest-lens lê lens_patents.jsonl e enriquece a tabela patents com dados Lens.org.
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

	"github.com/lib/pq"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	pg "github.com/LeoPani/agora/backend/internal/repository/postgres"
)

const dataDir = "../ai-service/data"

type lensPatent struct {
	LensID            string   `json:"lens_id"`
	INPINumber        string   `json:"inpi_number"`
	Title             string   `json:"title"`
	Abstract          string   `json:"abstract"`
	ApplicationNumber string   `json:"application_number"`
	FilingDate        string   `json:"filing_date"`
	PublicationDate   string   `json:"publication_date"`
	GrantDate         string   `json:"grant_date"`
	Applicants        []string `json:"applicants"`
	Inventors         []string `json:"inventors"`
	Jurisdiction      string   `json:"jurisdiction"`
	LegalStatus       string   `json:"legal_status"`
	IPCCodes          []string `json:"ipc_codes"`
	PatentCitations   []string `json:"patent_citations"`
	NPLCitations      []string `json:"npl_citations"`
	CitedByCount      int      `json:"cited_by_count"`
	FamilySize        int      `json:"family_size"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-lens: fatal", "err", err)
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
	runID, err := collectorRepo.StartRun(ctx, "ingest-lens")
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
	log.Info("ingest-lens complete", "total", total)
	return nil
}

func ingest(ctx context.Context, log *slog.Logger, db *sql.DB) (int, error) {
	path := filepath.Join(dataDir, "lens_patents.jsonl")
	patents, err := loadJSONL(path)
	if err != nil {
		return 0, err
	}
	log.Info("loaded lens patents", "count", len(patents))

	upserted := 0
	for _, p := range patents {
		if p.Title == "" && p.LensID == "" {
			continue
		}

		inpiNumber := p.INPINumber
		if inpiNumber == "" && strings.HasPrefix(p.ApplicationNumber, "BR") {
			inpiNumber = p.ApplicationNumber
		}

		var filingDate, pubDate, grantDate interface{}
		if p.FilingDate != "" {
			filingDate = p.FilingDate
		}
		if p.PublicationDate != "" {
			pubDate = p.PublicationDate
		}
		if p.GrantDate != "" {
			grantDate = p.GrantDate
		}

		inventorsJSON, _ := json.Marshal(p.Inventors)
		applicantsJSON, _ := json.Marshal(p.Applicants)
		citsJSON, _ := json.Marshal(p.NPLCitations)

		_, err := db.ExecContext(ctx, `
			INSERT INTO patents
			  (inpi_number, title, abstract, filing_date, publication_date, grant_date,
			   ipc_codes, inventors, status, is_ufv, raw_data)
			VALUES ($1, $2, $3, $4::DATE, $5::DATE, $6::DATE,
			        $7, $8::jsonb, $9, true,
			        jsonb_build_object(
			          'lens_id', $10,
			          'applicants', $11::jsonb,
			          'npl_citations', $12::jsonb,
			          'family_size', $13,
			          'cited_by_count', $14
			        ))
			ON CONFLICT (inpi_number) DO UPDATE SET
			  title            = COALESCE(EXCLUDED.title, patents.title),
			  abstract         = COALESCE(EXCLUDED.abstract, patents.abstract),
			  filing_date      = COALESCE(EXCLUDED.filing_date, patents.filing_date),
			  publication_date = COALESCE(EXCLUDED.publication_date, patents.publication_date),
			  grant_date       = COALESCE(EXCLUDED.grant_date, patents.grant_date),
			  ipc_codes        = CASE WHEN array_length(EXCLUDED.ipc_codes, 1) > 0
			                          THEN EXCLUDED.ipc_codes ELSE patents.ipc_codes END,
			  raw_data         = patents.raw_data || EXCLUDED.raw_data`,
			nullStr(inpiNumber),
			p.Title,
			nullStr(p.Abstract),
			filingDate,
			pubDate,
			grantDate,
			pq.Array(p.IPCCodes),
			string(inventorsJSON),
			nullStr(p.LegalStatus),
			p.LensID,
			string(applicantsJSON),
			string(citsJSON),
			p.FamilySize,
			p.CitedByCount,
		)
		if err != nil {
			log.Warn("upsert lens patent", "lens_id", p.LensID, "err", err)
			continue
		}
		upserted++
	}

	log.Info("lens ingest done", "upserted", upserted)
	return upserted, nil
}

func loadJSONL(path string) ([]lensPatent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var out []lensPatent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var p lensPatent
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
