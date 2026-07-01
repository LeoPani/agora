package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

// ingestTriggerHandler — chamado pelo py-scheduler após cada coleta Python.
// Executa o binário de ingest correspondente (compilado ou go run em dev).
//
// Em produção Docker os binários estão em /usr/local/bin/agora-ingest-*.
// Em dev, usa INGEST_BIN_DIR ou cai em `go run ./cmd/collectors/ingest-*`.
func ingestTriggerHandler() http.HandlerFunc {
	binDir := os.Getenv("INGEST_BIN_DIR") // /usr/local/bin em Docker, "" em dev

	// source → nome do binário compilado e path do go run
	type ingestDef struct {
		binary  string   // agora-ingest-openalex
		goRun   []string // go run ./cmd/collectors/ingest-openalex
		backDir string   // dir para go run (relativo ao CWD da API)
	}

	defs := map[string]ingestDef{
		"openalex":      {binary: "agora-ingest-openalex",      goRun: []string{"go", "run", "./cmd/collectors/ingest-openalex"}},
		"locus":         {binary: "agora-ingest-locus",         goRun: []string{"go", "run", "./cmd/collectors/ingest-locus"}},
		"dgp":           {binary: "agora-ingest-dgp",           goRun: []string{"go", "run", "./cmd/collectors/ingest-dgp"}},
		"opportunities": {binary: "agora-ingest-opportunities", goRun: []string{"go", "run", "./cmd/collectors/ingest-opportunities"}},
		"comex":         {binary: "agora-ingest-comex",         goRun: []string{"go", "run", "./cmd/collectors/ingest-comex"}},
		"trends":        {binary: "agora-ingest-trends",        goRun: []string{"go", "run", "./cmd/collectors/ingest-trends"}},
		"inpi":          {binary: "agora-ingest-inpi",          goRun: []string{"go", "run", "./cmd/collectors/ingest-inpi-dataset"}},
		"embeddings":    {binary: "agora-ingest-embeddings",    goRun: []string{"go", "run", "./cmd/collectors/ingest-embeddings"}},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		source := r.PathValue("source")
		def, ok := defs[source]
		if !ok {
			writeErr(w, http.StatusBadRequest, "unknown ingest source: "+source)
			return
		}

		var cmd *exec.Cmd
		if binDir != "" {
			// Docker: executa binário compilado
			binPath := filepath.Join(binDir, def.binary)
			cmd = exec.CommandContext(r.Context(), binPath)
		} else {
			// Dev: go run
			cmd = exec.CommandContext(r.Context(), def.goRun[0], def.goRun[1:]...)
		}

		slog.Info("ingest triggered", "source", source)
		out, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("ingest failed", "source", source, "err", err, "output", string(out))
			writeErr(w, http.StatusInternalServerError, "ingest failed: "+err.Error())
			return
		}

		slog.Info("ingest completed", "source", source)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "source": source})
	}
}
