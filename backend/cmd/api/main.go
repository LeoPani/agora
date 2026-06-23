// agora-api — servidor HTTP principal.
// Endpoints: /health, /api/v1/stats, /api/v1/collector-runs,
//            /api/v1/publications, /api/v1/patents, /api/v1/groups,
//            /api/v1/opportunities, /api/v1/import-gaps, /api/v1/trends
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/lib/pq"

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

	cors := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r)
		}
	}

	limitParam := func(r *http.Request, def int) int {
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
				return n
			}
		}
		return def
	}

	mux := http.NewServeMux()

	// ── /health ────────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "agora-api"})
	})

	// ── /api/v1/stats ──────────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/stats", cors(func(w http.ResponseWriter, r *http.Request) {
		var cResearchers, cPublications, cPatents, cGroups int64
		var cOpps, cGaps, cTrends, cRuns int64
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM researchers").Scan(&cResearchers)
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM publications").Scan(&cPublications)
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM patents").Scan(&cPatents)
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM research_groups").Scan(&cGroups)
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM opportunities").Scan(&cOpps)
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM import_gaps").Scan(&cGaps)
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM market_trends").Scan(&cTrends)
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM collector_runs").Scan(&cRuns)

		var lastCollected *string
		var nextCollection string
		var ts time.Time
		if db.QueryRowContext(r.Context(),
			`SELECT finished_at FROM collector_runs WHERE status='ok' ORDER BY finished_at DESC LIMIT 1`,
		).Scan(&ts) == nil {
			s := ts.Format(time.RFC3339)
			lastCollected = &s
			nextCollection = ts.AddDate(0, 0, 7).Format("2006-01-02")
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"researchers":     cResearchers,
			"publications":    cPublications,
			"patents":         cPatents,
			"research_groups": cGroups,
			"opportunities":   cOpps,
			"import_gaps":     cGaps,
			"market_trends":   cTrends,
			"collector_runs":  cRuns,
			"last_collected":  lastCollected,
			"next_collection": nextCollection,
		})
	}))

	// ── /api/v1/collector-runs ─────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/collector-runs", cors(func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r, 100)
		rows, err := db.QueryContext(r.Context(), `
			SELECT id, collector_name, started_at, finished_at, status, records_collected, error_message
			FROM collector_runs ORDER BY started_at DESC LIMIT $1`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type run struct {
			ID               int64   `json:"id"`
			CollectorName    string  `json:"collector_name"`
			StartedAt        string  `json:"started_at"`
			FinishedAt       *string `json:"finished_at"`
			Status           *string `json:"status"`
			RecordsCollected int     `json:"records_collected"`
			ErrorMessage     *string `json:"error_message"`
		}
		var result []run
		for rows.Next() {
			var ru run
			var finAt *time.Time
			var stat, errMsg *string
			if err := rows.Scan(&ru.ID, &ru.CollectorName, &ru.StartedAt, &finAt, &stat, &ru.RecordsCollected, &errMsg); err != nil {
				continue
			}
			if finAt != nil {
				s := finAt.Format(time.RFC3339)
				ru.FinishedAt = &s
			}
			ru.Status = stat
			ru.ErrorMessage = errMsg
			result = append(result, ru)
		}
		if result == nil {
			result = []run{}
		}
		json.NewEncoder(w).Encode(result)
	}))

	// ── /api/v1/publications ───────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/publications", cors(func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r, 200)
		q := r.URL.Query().Get("q")
		var rows interface{ Next() bool; Close() error; Scan(...interface{}) error }
		var queryErr error
		if q != "" {
			rows, queryErr = db.QueryContext(r.Context(), `
				SELECT id, openalex_id, doi, title, abstract, publication_year, type, cited_by_count
				FROM publications WHERE title ILIKE $1 ORDER BY cited_by_count DESC LIMIT $2`,
				"%"+q+"%", limit)
		} else {
			rows, queryErr = db.QueryContext(r.Context(), `
				SELECT id, openalex_id, doi, title, abstract, publication_year, type, cited_by_count
				FROM publications ORDER BY cited_by_count DESC LIMIT $1`, limit)
		}
		if queryErr != nil {
			http.Error(w, queryErr.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type pub struct {
			ID              int64   `json:"id"`
			OpenAlexID      *string `json:"openalex_id"`
			DOI             *string `json:"doi"`
			Title           string  `json:"title"`
			Abstract        *string `json:"abstract"`
			PublicationYear *int    `json:"publication_year"`
			Type            *string `json:"type"`
			CitedByCount    int     `json:"cited_by_count"`
		}
		var result []pub
		for rows.Next() {
			var p pub
			rows.Scan(&p.ID, &p.OpenAlexID, &p.DOI, &p.Title, &p.Abstract, &p.PublicationYear, &p.Type, &p.CitedByCount)
			result = append(result, p)
		}
		if result == nil {
			result = []pub{}
		}
		json.NewEncoder(w).Encode(result)
	}))

	// ── /api/v1/patents ────────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/patents", cors(func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r, 500)
		ufvOnly := r.URL.Query().Get("ufv") == "true"

		q := `SELECT id, inpi_number, title, abstract, filing_date, legal_status,
		             applicant_type = 'UFV' AS is_ufv, ipc_codes
		      FROM patents`
		if ufvOnly {
			q += " WHERE applicant_type = 'UFV'"
		}
		q += " ORDER BY filing_date DESC NULLS LAST LIMIT $1"

		rows, err := db.QueryContext(r.Context(), q, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type pat struct {
			ID         int64    `json:"id"`
			INPINumber *string  `json:"inpi_number"`
			Title      *string  `json:"title"`
			Abstract   *string  `json:"abstract"`
			FilingDate *string  `json:"filing_date"`
			Status     *string  `json:"status"`
			IsUFV      bool     `json:"is_ufv"`
			IPCCodes   []string `json:"ipc_codes"`
		}
		var result []pat
		for rows.Next() {
			var p pat
			var ipcArr []string
			rows.Scan(&p.ID, &p.INPINumber, &p.Title, &p.Abstract, &p.FilingDate,
				&p.Status, &p.IsUFV, pq.Array(&ipcArr))
			p.IPCCodes = ipcArr
			result = append(result, p)
		}
		if result == nil {
			result = []pat{}
		}
		json.NewEncoder(w).Encode(result)
	}))

	// ── /api/v1/groups ─────────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/groups", cors(func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r, 200)
		rows, err := db.QueryContext(r.Context(), `
			SELECT id, dgp_id, name, leader, department, research_lines, main_area, formation_year
			FROM research_groups ORDER BY name LIMIT $1`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type grp struct {
			ID            int64    `json:"id"`
			DGPID         *string  `json:"dgp_id"`
			Name          string   `json:"name"`
			Leader        *string  `json:"leader"`
			Department    *string  `json:"department"`
			ResearchLines []string `json:"research_lines"`
			MainArea      *string  `json:"main_area"`
			FormationYear *int     `json:"formation_year"`
		}
		var result []grp
		for rows.Next() {
			var g grp
			rows.Scan(&g.ID, &g.DGPID, &g.Name, &g.Leader, &g.Department,
				pq.Array(&g.ResearchLines), &g.MainArea, &g.FormationYear)
			result = append(result, g)
		}
		if result == nil {
			result = []grp{}
		}
		json.NewEncoder(w).Encode(result)
	}))

	// ── /api/v1/opportunities ──────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/opportunities", cors(func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r, 200)
		rows, err := db.QueryContext(r.Context(), `
			SELECT id, source, external_id, title, description, url, closing_date::text, status
			FROM opportunities ORDER BY collected_at DESC LIMIT $1`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type opp struct {
			ID         int64   `json:"id"`
			Source     string  `json:"source"`
			ExternalID string  `json:"external_id"`
			Title      string  `json:"title"`
			Desc       *string `json:"description"`
			URL        *string `json:"url"`
			Deadline   *string `json:"deadline"`
			Status     *string `json:"status"`
		}
		var result []opp
		for rows.Next() {
			var o opp
			rows.Scan(&o.ID, &o.Source, &o.ExternalID, &o.Title, &o.Desc, &o.URL, &o.Deadline, &o.Status)
			result = append(result, o)
		}
		if result == nil {
			result = []opp{}
		}
		json.NewEncoder(w).Encode(result)
	}))

	// ── /api/v1/import-gaps ────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/import-gaps", cors(func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r, 200)
		rows, err := db.QueryContext(r.Context(), `
			SELECT id, sh4_code, description, country_origin, year,
			       import_value_usd, import_kg, ufv_related_areas, opportunity_score
			FROM import_gaps ORDER BY opportunity_score DESC LIMIT $1`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type gap struct {
			ID               int64    `json:"id"`
			SH4Code          string   `json:"sh4_code"`
			Description      *string  `json:"description"`
			CountryOrigin    string   `json:"country_origin"`
			Year             int      `json:"year"`
			ImportValueUSD   float64  `json:"import_value_usd"`
			ImportKG         float64  `json:"import_kg"`
			UFVRelatedAreas  []string `json:"ufv_related_areas"`
			OpportunityScore float64  `json:"opportunity_score"`
		}
		var result []gap
		for rows.Next() {
			var g gap
			rows.Scan(&g.ID, &g.SH4Code, &g.Description, &g.CountryOrigin, &g.Year,
				&g.ImportValueUSD, &g.ImportKG, pq.Array(&g.UFVRelatedAreas), &g.OpportunityScore)
			result = append(result, g)
		}
		if result == nil {
			result = []gap{}
		}
		json.NewEncoder(w).Encode(result)
	}))

	// ── /api/v1/trends ─────────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/trends", cors(func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r, 200)
		rows, err := db.QueryContext(r.Context(), `
			SELECT id, keyword, geo, timeframe, avg_interest, peak_interest,
			       growth_pct, ufv_department
			FROM market_trends ORDER BY growth_pct DESC LIMIT $1`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type trend struct {
			ID            int64   `json:"id"`
			Keyword       string  `json:"keyword"`
			Geo           string  `json:"geo"`
			Timeframe     string  `json:"timeframe"`
			AvgInterest   int     `json:"avg_interest"`
			PeakInterest  int     `json:"peak_interest"`
			GrowthPct     float64 `json:"growth_pct"`
			UFVDepartment *string `json:"ufv_department"`
		}
		var result []trend
		for rows.Next() {
			var t trend
			rows.Scan(&t.ID, &t.Keyword, &t.Geo, &t.Timeframe, &t.AvgInterest,
				&t.PeakInterest, &t.GrowthPct, &t.UFVDepartment)
			result = append(result, t)
		}
		if result == nil {
			result = []trend{}
		}
		json.NewEncoder(w).Encode(result)
	}))

	srv := &http.Server{
		Addr:         cfg.APIAddr,
		Handler:      mux,
		ReadTimeout:  cfg.APIReadTimeout,
		WriteTimeout: cfg.APIWriteTimeout,
	}

	log.Info("agora-api starting", "addr", cfg.APIAddr)
	return srv.ListenAndServe()
}
