// agora-scheduler agenda e dispara coletas periódicas.
// Por enquanto tem placeholders para OpenAlex (mensal + incremental semanal).
// Futuramente: LOCUS, Lens, editais (FAPEMIG, FINEP, CNPq, EMBRAPII).
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
	cmd      []string
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

	jobs := []*job{
		{
			name:     "openalex-full",
			interval: 30 * 24 * time.Hour, // mensal
			cmd:      []string{"go", "run", "./cmd/collectors/ingest-openalex"},
		},
		{
			name:     "openalex-incremental",
			interval: 7 * 24 * time.Hour, // semanal
			cmd:      []string{"go", "run", "./cmd/collectors/ingest-openalex", "--incremental"},
		},
		// TODO: locus-dissertations   (semanal)
		// TODO: lens-citations        (mensal)
		// TODO: editais-fapemig       (semanal)
		// TODO: editais-finep         (semanal)
		// TODO: editais-cnpq          (semanal)
	}

	log.Info("scheduler started", "jobs", len(jobs))

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			for _, j := range jobs {
				if now.Sub(j.lastRun) >= j.interval {
					log.Info("running job", "name", j.name)
					j.lastRun = now
					go runJob(j, log)
				}
			}
		}
	}
}

func runJob(j *job, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	cmd := exec.CommandContext(ctx, j.cmd[0], j.cmd[1:]...)
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("job failed", "name", j.name, "err", err, "output", string(out))
		return
	}
	log.Info("job completed", "name", j.name)
}
