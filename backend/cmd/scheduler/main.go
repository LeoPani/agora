// agora-scheduler agenda e dispara coletas periódicas para todos os coletores.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
)

type job struct {
	name     string
	interval time.Duration
	// pyCmd: se não vazio, roda via Python
	pyScript string
	// goCmd: se não vazio, roda via go run
	goCmd   []string
	lastRun time.Time
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

	py := "../ai-service/venv/bin/python3"

	jobs := []*job{
		// OpenAlex — coleta mensal completa + ingestão
		{name: "openalex-collect", interval: 30 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/openalex_collector.py"},
		{name: "openalex-ingest", interval: 30 * 24 * time.Hour,
			goCmd: []string{"go", "run", "./cmd/collectors/ingest-openalex"}},

		// LOCUS — semanal (novo conteúdo DSpace é publicado com frequência)
		{name: "locus-collect", interval: 7 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/locus_collector.py"},
		{name: "locus-ingest", interval: 7 * 24 * time.Hour,
			goCmd: []string{"go", "run", "./cmd/collectors/ingest-locus"}},

		// DGP — mensal
		{name: "dgp-collect", interval: 30 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/dgp_collector.py"},
		{name: "dgp-ingest", interval: 30 * 24 * time.Hour,
			goCmd: []string{"go", "run", "./cmd/collectors/ingest-dgp"}},

		// Editais — semanal (prazos mudam frequentemente)
		{name: "fapemig-collect", interval: 7 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/fapemig_collector.py"},
		{name: "finep-collect", interval: 7 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/finep_collector.py"},
		{name: "cnpq-collect", interval: 7 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/cnpq_collector.py"},
		{name: "embrapii-collect", interval: 7 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/embrapii_collector.py"},
		{name: "opportunities-ingest", interval: 7 * 24 * time.Hour,
			goCmd: []string{"go", "run", "./cmd/collectors/ingest-opportunities"}},

		// Comex Stat — mensal (dados MDIC com lag de 30d)
		{name: "comex-collect", interval: 30 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/comex_collector.py"},
		{name: "comex-ingest", interval: 30 * 24 * time.Hour,
			goCmd: []string{"go", "run", "./cmd/collectors/ingest-comex"}},

		// Google Trends — quinzenal
		{name: "trends-collect", interval: 15 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/trends_collector.py"},
		{name: "trends-ingest", interval: 15 * 24 * time.Hour,
			goCmd: []string{"go", "run", "./cmd/collectors/ingest-trends"}},

		// INPI — semestral (dataset HuggingFace atualizado pontualmente)
		{name: "inpi-collect", interval: 180 * 24 * time.Hour,
			pyScript: "../ai-service/collectors/inpi_dataset_loader.py"},
		{name: "inpi-ingest", interval: 180 * 24 * time.Hour,
			goCmd: []string{"go", "run", "./cmd/collectors/ingest-inpi-dataset"}},
	}

	log.Info("scheduler started", "jobs", len(jobs))

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for _, j := range jobs {
			if now.Sub(j.lastRun) >= j.interval {
				log.Info("triggering job", "name", j.name)
				j.lastRun = now
				go runJob(j, py, log)
			}
		}
	}
	return nil
}

func runJob(j *job, py string, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	var cmd *exec.Cmd
	if j.pyScript != "" {
		cmd = exec.CommandContext(ctx, py, j.pyScript)
		cmd.Dir = "."
	} else {
		cmd = exec.CommandContext(ctx, j.goCmd[0], j.goCmd[1:]...)
		cmd.Dir = "."
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("job failed", "name", j.name, "err", err, "output", string(out))
		return
	}
	log.Info("job completed", "name", j.name)
}
