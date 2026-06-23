// agora-ingest-locus lê locus_publications.jsonl e upserta no Postgres.
// UPSERT por handle do LOCUS. Source = "LOCUS".
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
	"strings"
	"time"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/domain"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	pg "github.com/LeoPani/agora/backend/internal/repository/postgres"
)

const dataDir = "../ai-service/data"

type rawLocusPub struct {
	LocusUUID       string   `json:"locus_uuid"`
	Handle          string   `json:"handle"`
	Title           string   `json:"title"`
	Abstract        string   `json:"abstract"`
	Authors         []string `json:"authors"`
	Advisor         string   `json:"advisor"`
	PublicationYear int      `json:"publication_year"`
	Type            string   `json:"type"`
	Department      string   `json:"department"`
	Collection      string   `json:"collection"`
	URL             string   `json:"url"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-locus: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
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
	pubRepo       := pg.NewPublicationRepo(db)

	runID, err := collectorRepo.StartRun(ctx, "ingest-locus")
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}

	total, ingestErr := ingest(ctx, log, resRepo, pubRepo)

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
	log.Info("ingest-locus complete", "total", total)
	return nil
}

func ingest(ctx context.Context, log *slog.Logger, resRepo *pg.ResearcherRepo, pubRepo *pg.PublicationRepo) (int, error) {
	path := filepath.Join(dataDir, "locus_publications.jsonl")
	pubs, err := loadJSONL[rawLocusPub](path)
	if err != nil {
		return 0, fmt.Errorf("load locus_publications: %w", err)
	}
	log.Info("loaded", "file", path, "count", len(pubs))

	inserted, skipped := 0, 0

	for _, p := range pubs {
		if p.Title == "" {
			skipped++
			continue
		}

		// Usar locus_uuid como openalex_id surrogate com prefixo
		openAlexLike := "LOCUS:" + p.LocusUUID
		if p.LocusUUID == "" {
			openAlexLike = "LOCUS-HANDLE:" + p.Handle
		}

		// Limpa abstract muito longo
		abstract := p.Abstract
		if len(abstract) > 5000 {
			abstract = abstract[:5000]
		}

		topics, _ := json.Marshal([]string{p.Department, p.Type})

		pubID, err := pubRepo.UpsertWithSource(ctx, &domain.Publication{
			OpenAlexID:      openAlexLike,
			Title:           p.Title,
			Abstract:        abstract,
			PublicationYear: p.PublicationYear,
			Type:            p.Type,
			Topics:          topics,
		}, "LOCUS")
		if err != nil {
			if errors.Is(err, domain.ErrDuplicate) {
				skipped++
				continue
			}
			log.Warn("upsert pub", "title", p.Title[:min(40, len(p.Title))], "err", err)
			continue
		}

		// Upsert autores
		for pos, authorName := range p.Authors {
			if authorName == "" {
				continue
			}
			norm := normalizeAuthor(authorName)
			resID, err := resRepo.Upsert(ctx, &domain.Researcher{
				FullName:       authorName,
				NormalizedName: norm,
				Institution:    "UFV",
				Department:     inferDept(p.Department),
			})
			if err != nil && !errors.Is(err, domain.ErrDuplicate) {
				continue
			}
			if errors.Is(err, domain.ErrDuplicate) {
				resID, _ = resRepo.FindByNormalizedName(ctx, norm)
			}
			if resID > 0 {
				_ = pubRepo.LinkAuthor(ctx, &domain.PublicationAuthor{
					PublicationID: pubID, ResearcherID: resID, AuthorPosition: pos + 1,
				})
			}
		}

		inserted++
		if inserted%500 == 0 {
			log.Info("progress", "inserted", inserted)
		}
	}

	log.Info("locus ingest done", "inserted", inserted, "skipped", skipped)
	return inserted, nil
}

func normalizeAuthor(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	// Remove sufixos comuns de citação (ex: "SILVA, João" → "joao silva")
	if idx := strings.Index(name, ","); idx > 0 && idx < 30 {
		last := strings.TrimSpace(name[:idx])
		first := strings.TrimSpace(name[idx+1:])
		name = first + " " + last
	}
	// Simplifica: remove acentos ASCII approximation
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func inferDept(communityName string) string {
	name := strings.ToLower(communityName)
	switch {
	case strings.Contains(name, "alimentos"):
		return "DTA"
	case strings.Contains(name, "fitotecnia"):
		return "DFT"
	case strings.Contains(name, "fitopatologia"):
		return "DFP"
	case strings.Contains(name, "informática") || strings.Contains(name, "computação"):
		return "DPI"
	case strings.Contains(name, "química"):
		return "DQI"
	case strings.Contains(name, "bioquímica") || strings.Contains(name, "biologia molecular"):
		return "DBB"
	case strings.Contains(name, "florestal"):
		return "DEF"
	case strings.Contains(name, "engenharia agrícola"):
		return "DEA"
	}
	return ""
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
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		out = append(out, item)
	}
	return out, scanner.Err()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
