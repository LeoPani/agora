// agora-ingest-inpi-dataset lê inpi_patents_part_*.jsonl e upserta na tabela patents.
// Processa em batches de 5000, registra em collector_runs.
// Patentes UFV (is_ufv=true) são linkadas a researchers existentes por nome.
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
	"strings"
	"time"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	pg "github.com/LeoPani/agora/backend/internal/repository/postgres"
)

const (
	dataDir   = "../ai-service/data"
	batchSize = 5000
)

type rawPatent struct {
	INPINumber    string   `json:"inpi_number"`
	Title         string   `json:"title"`
	Abstract      string   `json:"abstract"`
	IPCCode       string   `json:"ipc_code"`
	IPCSection    string   `json:"ipc_section"`
	FilingDate    string   `json:"filing_date"`
	Applicant     string   `json:"applicant"`
	ApplicantType string   `json:"applicant_type"`
	IsUFV         bool     `json:"is_ufv"`
	Inventors     []string `json:"inventors"`
	RawData       any      `json:"raw_data"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-inpi-dataset: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Minute)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		DSN:             cfg.DatabaseURL,
		MaxOpenConns:    8,
		MaxIdleConns:    4,
		ConnMaxLifetime: 10 * time.Minute,
		PingTimeout:     5 * time.Second,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	collectorRepo := pg.NewCollectorRepo(db)
	resRepo       := pg.NewResearcherRepo(db)

	runID, err := collectorRepo.StartRun(ctx, "ingest-inpi-dataset")
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}

	total, ingestErr := ingest(ctx, log, db, resRepo)

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
	log.Info("ingest-inpi-dataset complete", "total", total)
	return nil
}

func ingest(ctx context.Context, log *slog.Logger, db *sql.DB, resRepo *pg.ResearcherRepo) (int, error) {
	// Listar arquivos part_*.jsonl em ordem
	pattern := filepath.Join(dataDir, "inpi_patents_part_*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return 0, fmt.Errorf("nenhum arquivo encontrado em %s", pattern)
	}
	sort.Strings(files)
	log.Info("files found", "count", len(files))

	total, inserted, skipped, ufvLinked := 0, 0, 0, 0

	for _, file := range files {
		log.Info("processing file", "path", file)
		patents, err := loadJSONLFile(file)
		if err != nil {
			log.Warn("load file failed", "file", file, "err", err)
			continue
		}

		// Processar em batches dentro de transação
		for start := 0; start < len(patents); start += batchSize {
			end := start + batchSize
			if end > len(patents) {
				end = len(patents)
			}
			batch := patents[start:end]

			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				log.Warn("begin tx", "err", err)
				continue
			}

			for _, p := range batch {
				if p.INPINumber == "" || p.Title == "" {
					skipped++
					continue
				}

				rawJSON, _ := json.Marshal(p.RawData)
				ipcCodes := "ARRAY[]::TEXT[]"
				if p.IPCCode != "" {
					ipcCodes = fmt.Sprintf("ARRAY['%s']", strings.ReplaceAll(p.IPCCode, "'", "''"))
				}

				var ipcSection *string
				if len(p.IPCSection) >= 1 {
					s := strings.ToUpper(string([]rune(p.IPCSection)[0]))
					ipcSection = &s
				}

				var filingDate *string
				if p.FilingDate != "" {
					fd := normalizeDate(p.FilingDate)
					if fd != "" {
						filingDate = &fd
					}
				}

				var id int64
				q := fmt.Sprintf(`
					INSERT INTO patents (inpi_number, title, abstract, ipc_codes, ipc_section,
					                    filing_date, applicant, applicant_type, raw_data)
					VALUES ($1, $2, $3, %s, $4, $5, $6, $7, $8)
					ON CONFLICT (inpi_number) DO UPDATE SET
						title = EXCLUDED.title,
						raw_data = EXCLUDED.raw_data
					RETURNING id`, ipcCodes)

				err := tx.QueryRowContext(ctx, q,
					p.INPINumber,
					p.Title,
					nullStr(p.Abstract),
					nullStr1(ipcSection),
					nullStr(dateVal(filingDate)),
					p.Applicant,
					nullStr(p.ApplicantType),
					rawJSON,
				).Scan(&id)

				if err != nil {
					skipped++
					continue
				}
				inserted++

				// Link inventors para patentes UFV
				if p.IsUFV && id > 0 {
					for _, inv := range p.Inventors {
						if inv == "" {
							continue
						}
						norm := strings.ToLower(strings.TrimSpace(inv))
						resID, err := resRepo.FindByNormalizedName(ctx, norm)
						if err != nil || resID == 0 {
							continue
						}
						_, _ = tx.ExecContext(ctx,
							`INSERT INTO patent_inventors (patent_id, researcher_id)
							 VALUES ($1, $2) ON CONFLICT DO NOTHING`,
							id, resID)
						ufvLinked++
					}
				}
			}

			if err := tx.Commit(); err != nil {
				log.Warn("commit batch", "err", err)
				_ = tx.Rollback()
			}

			total += len(batch)
			if total%50000 == 0 {
				log.Info("progress", "total", total, "inserted", inserted)
			}
		}
	}

	log.Info("ingest-inpi done",
		"total_processed", total,
		"inserted", inserted,
		"skipped", skipped,
		"ufv_inventor_links", ufvLinked,
	)
	return inserted, nil
}

func loadJSONLFile(path string) ([]rawPatent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []rawPatent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var p rawPatent
		if err := json.Unmarshal(line, &p); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, scanner.Err()
}

func normalizeDate(s string) string {
	// Aceita YYYY-MM-DD ou DD/MM/YYYY
	s = strings.TrimSpace(s)
	if len(s) == 10 && s[4] == '-' {
		return s
	}
	if len(s) == 10 && s[2] == '/' {
		parts := strings.Split(s, "/")
		if len(parts) == 3 {
			return parts[2] + "-" + parts[1] + "-" + parts[0]
		}
	}
	return ""
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullStr1(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func dateVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
