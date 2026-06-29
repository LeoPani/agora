// agora-ingest-dgp lê dgp_research_groups.jsonl e popula research_groups + research_group_members.
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

var dataDir = config.DataDir()

type rawGroup struct {
	DGPID         string   `json:"dgp_id"`
	Name          string   `json:"name"`
	Leader        string   `json:"leader"`
	Department    string   `json:"department"`
	ResearchLines []string `json:"research_lines"`
	MainArea      string   `json:"main_area"`
	FormationYear *int     `json:"formation_year"`
	Institution   string   `json:"institution"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-dgp: fatal", "err", err)
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
	resRepo       := pg.NewResearcherRepo(db)
	runID, err := collectorRepo.StartRun(ctx, "ingest-dgp")
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
	log.Info("ingest-dgp complete", "total", total)
	return nil
}

func ingest(ctx context.Context, log *slog.Logger, db *sql.DB, resRepo *pg.ResearcherRepo) (int, error) {
	path := filepath.Join(dataDir, "dgp_research_groups.jsonl")
	groups, err := loadJSONL(path)
	if err != nil {
		return 0, err
	}
	log.Info("loaded groups", "count", len(groups))

	inserted := 0
	for _, g := range groups {
		if g.Name == "" {
			continue
		}

		linesJSON, _ := json.Marshal(g.ResearchLines)
		dgpID := g.DGPID
		if dgpID == "" {
			dgpID = "UFV-" + strings.ToUpper(strings.ReplaceAll(g.Name[:min(20, len(g.Name))], " ", "-"))
		}

		var groupID int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO research_groups
			  (dgp_id, name, leader, department, research_lines, main_area, formation_year, institution)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (dgp_id) DO UPDATE SET
			  name = EXCLUDED.name,
			  leader = EXCLUDED.leader
			RETURNING id`,
			dgpID,
			g.Name,
			nullStr(g.Leader),
			nullStr(g.Department),
			pq.Array(g.ResearchLines),
			nullStr(g.MainArea),
			g.FormationYear,
			stringOr(g.Institution, "UFV"),
		).Scan(&groupID)
		if err != nil {
			log.Warn("upsert group", "name", g.Name, "err", err)
			continue
		}

		// Tentar linkar o líder como researcher
		if g.Leader != "" {
			norm := strings.ToLower(strings.TrimSpace(g.Leader))
			resID, err := resRepo.FindByNormalizedName(ctx, norm)
			if err == nil && resID > 0 {
				_, _ = db.ExecContext(ctx, `
					INSERT INTO research_group_members (group_id, researcher_id, role)
					VALUES ($1, $2, 'lider') ON CONFLICT DO NOTHING`, groupID, resID)
			}
		}

		_ = linesJSON
		inserted++
	}

	log.Info("ingest-dgp done", "inserted", inserted)
	return inserted, nil
}

func loadJSONL(path string) ([]rawGroup, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var out []rawGroup
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var g rawGroup
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

func stringOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
