// agora-api é o servidor HTTP principal do Ágora.
// Por enquanto expõe apenas /health e /api/v1/stats (contagens básicas).
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})

	ctx := context.Background()
	db, err := database.New(ctx, database.Config{
		DSN:             cfg.DatabaseURL,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		PingTimeout:     5 * time.Second,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "agora-api"})
	})

	mux.HandleFunc("GET /api/v1/stats", func(w http.ResponseWriter, r *http.Request) {
		var researchers, publications int
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM researchers").Scan(&researchers)
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM publications").Scan(&publications)

		var lastCollected *string
		var nextCollection string
		row := db.QueryRowContext(r.Context(),
			`SELECT finished_at FROM collector_runs
			 WHERE status = 'ok' ORDER BY finished_at DESC LIMIT 1`)
		var ts time.Time
		if row.Scan(&ts) == nil {
			s := ts.Format("2006-01-02T15:04:05Z")
			lastCollected = &s
			next := ts.AddDate(0, 0, 7)
			nextCollection = next.Format("2006-01-02")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"researchers":     researchers,
			"publications":    publications,
			"last_collected":  lastCollected,
			"next_collection": nextCollection,
		})
	})

	srv := &http.Server{
		Addr:         cfg.APIAddr,
		Handler:      mux,
		ReadTimeout:  cfg.APIReadTimeout,
		WriteTimeout: cfg.APIWriteTimeout,
	}

	log.Info("agora-api starting", "addr", cfg.APIAddr)
	return srv.ListenAndServe()
}
