// agora-api — servidor HTTP principal.
// Endpoints: /health, /api/v1/stats, /api/v1/collector-runs,
//            /api/v1/publications, /api/v1/patents, /api/v1/groups,
//            /api/v1/opportunities, /api/v1/import-gaps, /api/v1/trends,
//            /api/v1/search (busca semântica via pgvector)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"

	"github.com/joho/godotenv"

	"github.com/LeoPani/agora/backend/internal/config"
	"github.com/LeoPani/agora/backend/internal/llm"
	"github.com/LeoPani/agora/backend/internal/platform/database"
	"github.com/LeoPani/agora/backend/internal/platform/logger"
	"github.com/LeoPani/agora/backend/internal/rag"
)

func main() {
	// Carrega .env — tenta CWD e depois o diretório do executável
	if err := godotenv.Load(".env"); err != nil {
		if exe, err2 := os.Executable(); err2 == nil {
			godotenv.Load(filepath.Dir(exe) + "/.env")
		}
	}
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

	// ── /api/v1/partners ───────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/partners", cors(func(w http.ResponseWriter, r *http.Request) {
		limit      := limitParam(r, 200)
		src        := r.URL.Query().Get("source")
		ptype      := r.URL.Query().Get("type")
		q          := r.URL.Query().Get("q")

		query := `SELECT id, name, cnpj, partner_type, sector, location,
		                 cnae_code, lattes_id, linkedin_url, contact_email,
		                 interest_score, source, n_citations_to_ufv
		          FROM partners WHERE 1=1`
		args := []interface{}{}
		n := 1

		if src != "" {
			query += fmt.Sprintf(" AND source = $%d", n)
			args = append(args, src); n++
		}
		if ptype != "" {
			query += fmt.Sprintf(" AND partner_type = $%d", n)
			args = append(args, ptype); n++
		}
		if q != "" {
			query += fmt.Sprintf(" AND (name ILIKE $%d OR sector ILIKE $%d)", n, n)
			args = append(args, "%"+q+"%"); n++
		}
		query += fmt.Sprintf(" ORDER BY interest_score DESC LIMIT $%d", n)
		args = append(args, limit)

		rows, err := db.QueryContext(r.Context(), query, args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type partner struct {
			ID             int64   `json:"id"`
			Name           string  `json:"name"`
			CNPJ           *string `json:"cnpj"`
			PartnerType    *string `json:"partner_type"`
			Sector         *string `json:"sector"`
			Location       *string `json:"location"`
			CNAECode       *string `json:"cnae_code"`
			LattesID       *string `json:"lattes_id"`
			LinkedInURL    *string `json:"linkedin_url"`
			ContactEmail   *string `json:"contact_email"`
			InterestScore  float64 `json:"interest_score"`
			Source         *string `json:"source"`
			NCitationsUFV  int     `json:"n_citations_to_ufv"`
		}
		var result []partner
		for rows.Next() {
			var p partner
			rows.Scan(&p.ID, &p.Name, &p.CNPJ, &p.PartnerType, &p.Sector, &p.Location,
				&p.CNAECode, &p.LattesID, &p.LinkedInURL, &p.ContactEmail,
				&p.InterestScore, &p.Source, &p.NCitationsUFV)
			result = append(result, p)
		}
		if result == nil {
			result = []partner{}
		}
		json.NewEncoder(w).Encode(result)
	}))

	// ── /api/v1/search ────────────────────────────────────────────────────────
	// Busca semântica via pgvector — requer embeddings gerados.
	// Parâmetros: ?q=texto&type=all|publications|patents&limit=20
	mux.HandleFunc("GET /api/v1/search", cors(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			http.Error(w, `{"error":"q is required"}`, http.StatusBadRequest)
			return
		}
		entityType := r.URL.Query().Get("type")
		if entityType == "" {
			entityType = "all"
		}
		limit := limitParam(r, 20)

		// Chama função de embedding via endpoint interno do ai-service
		embURL := "http://localhost:8082/embed?text=" + q
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(embURL)
		if err != nil || resp.StatusCode != 200 {
			// Fallback: busca lexical quando embeddings não disponíveis
			type result struct {
				ID     int64   `json:"id"`
				Title  string  `json:"title"`
				Score  float64 `json:"score"`
				Source string  `json:"source"`
				Type   string  `json:"type"`
			}
			var results []result
			likeQ := "%" + q + "%"
			if entityType == "all" || entityType == "publications" {
				rows, _ := db.QueryContext(r.Context(), `
					SELECT id, title, 0.5 as score, source, 'publication' as type
					FROM publications
					WHERE title ILIKE $1 OR abstract ILIKE $1
					ORDER BY cited_by_count DESC LIMIT $2`, likeQ, limit)
				if rows != nil {
					defer rows.Close()
					for rows.Next() {
						var res result
						rows.Scan(&res.ID, &res.Title, &res.Score, &res.Source, &res.Type)
						results = append(results, res)
					}
				}
			}
			if entityType == "all" || entityType == "patents" {
				rows, _ := db.QueryContext(r.Context(), `
					SELECT id, title, 0.5 as score, 'inpi' as source, 'patent' as type
					FROM patents
					WHERE title ILIKE $1 OR abstract ILIKE $1
					ORDER BY id DESC LIMIT $2`, likeQ, limit)
				if rows != nil {
					defer rows.Close()
					for rows.Next() {
						var res result
						rows.Scan(&res.ID, &res.Title, &res.Score, &res.Source, &res.Type)
						results = append(results, res)
					}
				}
			}
			if results == nil {
				results = []result{}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"mode":    "lexical",
				"query":   q,
				"results": results,
			})
			return
		}
		defer resp.Body.Close()

		var embData struct {
			Embedding []float32 `json:"embedding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&embData); err != nil || len(embData.Embedding) == 0 {
			http.Error(w, `{"error":"embedding decode failed"}`, http.StatusInternalServerError)
			return
		}
		vec := pgvector.NewVector(embData.Embedding)

		type result struct {
			ID       int64   `json:"id"`
			Title    string  `json:"title"`
			Abstract *string `json:"abstract,omitempty"`
			Score    float64 `json:"score"`
			Source   string  `json:"source"`
			Type     string  `json:"type"`
		}
		var results []result

		if entityType == "all" || entityType == "publications" {
			rows, err := db.QueryContext(r.Context(), `
				SELECT id, title, abstract, source,
				       1 - (embedding <=> $1) AS score
				FROM publications
				WHERE embedding IS NOT NULL
				ORDER BY embedding <=> $1
				LIMIT $2`, vec, limit)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var res result
					res.Type = "publication"
					rows.Scan(&res.ID, &res.Title, &res.Abstract, &res.Source, &res.Score)
					results = append(results, res)
				}
			}
		}
		if entityType == "all" || entityType == "patents" {
			rows, err := db.QueryContext(r.Context(), `
				SELECT id, title, abstract, 'inpi' as source,
				       1 - (embedding <=> $1) AS score
				FROM patents
				WHERE embedding IS NOT NULL
				ORDER BY embedding <=> $1
				LIMIT $2`, vec, limit)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var res result
					res.Type = "patent"
					rows.Scan(&res.ID, &res.Title, &res.Abstract, &res.Source, &res.Score)
					results = append(results, res)
				}
			}
		}
		if results == nil {
			results = []result{}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"mode":    "semantic",
			"query":   q,
			"results": results,
		})
	}))

	// ── AI Layer ───────────────────────────────────────────────────────────────
	llmCfg    := llm.LoadConfig()
	llmRouter := llm.NewRouter(llmCfg)
	llmLogger := llm.NewDBLogger(db, llmCfg.LogPath)
	embedURL := os.Getenv("EMBED_SERVER_URL")
	if embedURL == "" {
		embedURL = "http://localhost:8082"
	}
	retriever := rag.New(db, embedURL)

	mux.HandleFunc("POST /internal/llm/complete",          llmCompleteHandler(llmRouter, llmLogger))
	mux.HandleFunc("GET /api/v1/llm-stats",                llmStatsHandler(db))
	mux.HandleFunc("POST /api/chat",                       chatHandler(db, retriever, llmRouter, llmLogger))
	mux.HandleFunc("GET /api/conversations",               conversationsHandler(db))
	mux.HandleFunc("GET /api/conversations/{id}/messages", conversationMessagesHandler(db))
	mux.HandleFunc("GET /api/v1/departments",               departmentsHandler(db))
	mux.HandleFunc("GET /api/v1/matchmaking",               matchmakingHandler(db))
	mux.HandleFunc("POST /api/v1/matchmaking/compute",      computeMatchmakingHandler(db))
	mux.HandleFunc("PATCH /api/v1/matchmaking/{id}",        patchMatchHandler(db))
	mux.HandleFunc("GET /api/v1/signals",                   signalsHandler(db))
	mux.HandleFunc("GET /api/v1/agent-drafts",             agentDraftsHandler(db))
	mux.HandleFunc("PATCH /api/v1/agent-drafts/{id}",      patchDraftHandler(db))
	mux.HandleFunc("POST /api/v1/agent-drafts/generate",   generateDraftHandler(db, llmRouter, llmLogger))

	// ── /api/v1/linkedin-leads ─────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v1/linkedin-leads", cors(func(w http.ResponseWriter, r *http.Request) {
		leadsPath := "../ai-service/data/linkedin_search_leads.json"
		data, err := os.ReadFile(leadsPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Rode: make collect-linkedin"})
			return
		}
		w.Write(data)
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
