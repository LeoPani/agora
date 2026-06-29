package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/LeoPani/agora/backend/internal/llm"
)

// POST /internal/llm/complete — endpoint interno usado pelos workers Python.
func llmCompleteHandler(router *llm.Router, logger *llm.DBLogger) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}

		var req struct {
			Purpose      string         `json:"purpose"`
			Prompt       string         `json:"prompt"`
			Temperature  float64        `json:"temperature"`
			MaxTokens    int            `json:"max_tokens"`
			JSONMode     bool           `json:"json_mode"`
			ProviderHint string         `json:"provider_hint"`
			Metadata     map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if req.Purpose == "" {
			req.Purpose = "generic"
		}

		completionReq := llm.CompletionRequest{
			Purpose:     req.Purpose,
			Prompt:      req.Prompt,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			JSONMode:    req.JSONMode,
			Provider:    llm.Provider(req.ProviderHint),
			Metadata:    req.Metadata,
		}

		resp, err := router.Complete(r.Context(), completionReq)
		if logger != nil {
			logger.Log(r.Context(), completionReq, resp, err)
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"text":          resp.Text,
			"prompt_tokens": resp.PromptTokens,
			"output_tokens": resp.OutputTokens,
			"cost_usd":      resp.CostUSD,
			"latency_ms":    resp.LatencyMs,
			"provider":      resp.Provider,
			"model":         resp.Model,
		})
	})
}

// GET /api/v1/llm-stats — estatísticas de uso de LLM.
func llmStatsHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		var todayCost, monthCost float64
		var totalCalls, avgLatency int
		var errorRate float64

		db.QueryRowContext(r.Context(), `
			SELECT COALESCE(SUM(cost_usd),0), COUNT(*),
			       COALESCE(AVG(latency_ms),0),
			       COALESCE(SUM(CASE WHEN NOT success THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*),0), 0)
			FROM llm_calls
			WHERE created_at >= CURRENT_DATE
		`).Scan(&todayCost, &totalCalls, &avgLatency, &errorRate)

		db.QueryRowContext(r.Context(), `
			SELECT COALESCE(SUM(cost_usd),0)
			FROM llm_calls
			WHERE created_at >= DATE_TRUNC('month', NOW())
		`).Scan(&monthCost)

		// Purpose breakdown
		purposeRows, _ := db.QueryContext(r.Context(), `
			SELECT purpose, COUNT(*), COALESCE(SUM(cost_usd),0)
			FROM llm_calls
			WHERE created_at >= NOW() - INTERVAL '30 days'
			GROUP BY purpose ORDER BY SUM(cost_usd) DESC LIMIT 10`)
		type purposeStat struct {
			Purpose string  `json:"purpose"`
			Calls   int     `json:"calls"`
			CostUSD float64 `json:"cost_usd"`
		}
		var byPurpose []purposeStat
		if purposeRows != nil {
			defer purposeRows.Close()
			for purposeRows.Next() {
				var p purposeStat
				purposeRows.Scan(&p.Purpose, &p.Calls, &p.CostUSD)
				byPurpose = append(byPurpose, p)
			}
		}
		if byPurpose == nil {
			byPurpose = []purposeStat{}
		}

		// Recent calls
		recentRows, _ := db.QueryContext(r.Context(), `
			SELECT id, purpose, provider, model, total_tokens, cost_usd, latency_ms,
			       success, COALESCE(error_message,''), created_at
			FROM llm_calls ORDER BY created_at DESC LIMIT 50`)
		type callRow struct {
			ID        int64   `json:"id"`
			Purpose   string  `json:"purpose"`
			Provider  string  `json:"provider"`
			Model     string  `json:"model"`
			Tokens    int     `json:"total_tokens"`
			CostUSD   float64 `json:"cost_usd"`
			Latency   int     `json:"latency_ms"`
			Success   bool    `json:"success"`
			ErrMsg    string  `json:"error_message,omitempty"`
			CreatedAt string  `json:"created_at"`
		}
		var recent []callRow
		if recentRows != nil {
			defer recentRows.Close()
			for recentRows.Next() {
				var c callRow
				recentRows.Scan(&c.ID, &c.Purpose, &c.Provider, &c.Model,
					&c.Tokens, &c.CostUSD, &c.Latency, &c.Success, &c.ErrMsg, &c.CreatedAt)
				recent = append(recent, c)
			}
		}
		if recent == nil {
			recent = []callRow{}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"today_cost_usd":  todayCost,
			"month_cost_usd":  monthCost,
			"total_calls":     totalCalls,
			"avg_latency_ms":  avgLatency,
			"error_rate":      errorRate,
			"by_purpose":      byPurpose,
			"recent_calls":    recent,
		})
	})
}
