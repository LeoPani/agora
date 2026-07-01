// agora-scheduler agenda e dispara os ingest jobs Go periodicamente.
//
// Em Docker, os collectors Python são rodados pelo py-scheduler (ai-service/scheduler.py)
// e disparam ingest via POST /internal/ingest/:source na API.
// O scheduler Go roda apenas jobs que precisam de binários Go (ingest direto, sem coletor Python prévio).
//
// Env vars:
//   INGEST_BIN_DIR — "/usr/local/bin" em Docker, "" em dev (usa go run)
//   PYTHON_BIN     — path do Python; se vazio, jobs Python são ignorados (Docker)
//   COLLECTORS_DIR — dir dos collectors Python (default: ../ai-service/collectors)
//   BACKEND_DIR    — dir para go run (default: .)
package main

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type job struct {
	name     string
	interval time.Duration
	pyScript string   // relativo a COLLECTORS_DIR; vazio = job Go puro
	ingest   []string // args para o comando de ingest Go
	lastRun  time.Time
}

func main() {
	if err := run(); err != nil {
		slog.Error("scheduler: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})

	pyBin        := os.Getenv("PYTHON_BIN")        // vazio em Docker
	collectorsDir := envOr("COLLECTORS_DIR", "../ai-service/collectors")
	ingestBinDir  := os.Getenv("INGEST_BIN_DIR")   // "/usr/local/bin" em Docker
	backendDir    := envOr("BACKEND_DIR", ".")

	inDocker := ingestBinDir != "" // proxy para detectar ambiente Docker

	jobs := []*job{
		// OpenAlex — mensal (coleta + ingest em sequência)
		{name: "openalex-collect", interval: 30 * 24 * time.Hour,
			pyScript: "openalex_collector.py"},
		{name: "openalex-ingest", interval: 30 * 24 * time.Hour,
			ingest: ingestArgs("ingest-openalex", ingestBinDir)},

		// LOCUS — semanal
		{name: "locus-collect", interval: 7 * 24 * time.Hour,
			pyScript: "locus_collector.py"},
		{name: "locus-ingest", interval: 7 * 24 * time.Hour,
			ingest: ingestArgs("ingest-locus", ingestBinDir)},

		// DGP — mensal
		{name: "dgp-collect", interval: 30 * 24 * time.Hour,
			pyScript: "dgp_collector.py"},
		{name: "dgp-ingest", interval: 30 * 24 * time.Hour,
			ingest: ingestArgs("ingest-dgp", ingestBinDir)},

		// Editais — semanal
		{name: "editais-ingest", interval: 7 * 24 * time.Hour,
			ingest: ingestArgs("ingest-opportunities", ingestBinDir)},

		// Comex Stat — mensal
		{name: "comex-collect", interval: 30 * 24 * time.Hour,
			pyScript: "comex_collector.py"},
		{name: "comex-ingest", interval: 30 * 24 * time.Hour,
			ingest: ingestArgs("ingest-comex", ingestBinDir)},

		// Google Trends — quinzenal
		{name: "trends-collect", interval: 15 * 24 * time.Hour,
			pyScript: "trends_collector.py"},
		{name: "trends-ingest", interval: 15 * 24 * time.Hour,
			ingest: ingestArgs("ingest-trends", ingestBinDir)},

		// INPI — semestral
		{name: "inpi-collect", interval: 180 * 24 * time.Hour,
			pyScript: "inpi_dataset_loader.py"},
		{name: "inpi-ingest", interval: 180 * 24 * time.Hour,
			ingest: ingestArgs("ingest-inpi-dataset", ingestBinDir)},
	}

	log.Info("scheduler started", "jobs", len(jobs), "docker", inDocker)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for _, j := range jobs {
			if now.Sub(j.lastRun) < j.interval {
				continue
			}
			j.lastRun = now
			// Em Docker, jobs Python são delegados ao py-scheduler — ignorar
			if inDocker && j.pyScript != "" {
				continue
			}
			log.Info("triggering job", "name", j.name)
			go runJob(j, pyBin, collectorsDir, backendDir, log)
		}
	}
	return nil
}

// ingestArgs retorna o slice de args para o comando de ingest.
// Em Docker usa o binário compilado; em dev usa go run.
func ingestArgs(cmdSuffix, binDir string) []string {
	if binDir != "" {
		// binário compilado: agora-ingest-openalex etc.
		binName := "agora-" + cmdSuffix
		return []string{filepath.Join(binDir, binName)}
	}
	// dev: go run ./cmd/collectors/ingest-openalex
	return []string{"go", "run", "./cmd/collectors/" + cmdSuffix}
}

func runJob(j *job, pyBin, collectorsDir, backendDir string, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	var cmd *exec.Cmd
	if j.pyScript != "" {
		if pyBin == "" {
			log.Warn("skipping python job (PYTHON_BIN not set)", "name", j.name)
			return
		}
		scriptPath := filepath.Join(collectorsDir, j.pyScript)
		cmd = exec.CommandContext(ctx, pyBin, scriptPath)
		cmd.Dir = "."
	} else {
		// Go ingest
		if len(j.ingest) == 1 {
			// caminho absoluto para binário compilado
			cmd = exec.CommandContext(ctx, j.ingest[0])
		} else {
			cmd = exec.CommandContext(ctx, j.ingest[0], j.ingest[1:]...)
			cmd.Dir = backendDir
		}
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("job failed", "name", j.name, "err", err, "output", string(out))
		return
	}
	log.Info("job completed", "name", j.name)
}
