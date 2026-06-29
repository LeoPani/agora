package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/LeoPani/agora/backend/internal/agents"
	"github.com/LeoPani/agora/backend/internal/llm"
)

// GET /api/v1/signals
func signalsHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `
			SELECT id, signal_type, title, COALESCE(description,''), score,
			       COALESCE(entities::text,'{}'), status, created_at
			FROM signals ORDER BY score DESC, created_at DESC LIMIT 100`)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type sig struct {
			ID          int64           `json:"id"`
			Type        string          `json:"signal_type"`
			Title       string          `json:"title"`
			Description string          `json:"description"`
			Score       float64         `json:"score"`
			Entities    json.RawMessage `json:"entities"`
			Status      string          `json:"status"`
			CreatedAt   string          `json:"created_at"`
		}
		var result []sig
		for rows.Next() {
			var s sig
			var entJSON string
			rows.Scan(&s.ID, &s.Type, &s.Title, &s.Description, &s.Score,
				&entJSON, &s.Status, &s.CreatedAt)
			s.Entities = json.RawMessage(entJSON)
			result = append(result, s)
		}
		if result == nil {
			result = []sig{}
		}
		writeJSON(w, http.StatusOK, result)
	})
}

// GET /api/v1/agent-drafts
func agentDraftsHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `
			SELECT d.id, d.signal_id, s.title as signal_title,
			       d.draft_type, COALESCE(d.subject,''), d.body,
			       COALESCE(d.context_used::text,'{}'),
			       d.status, COALESCE(d.cost_usd, 0), d.created_at
			FROM agent_drafts d
			LEFT JOIN signals s ON s.id = d.signal_id
			ORDER BY d.created_at DESC LIMIT 100`)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type draft struct {
			ID           int64           `json:"id"`
			SignalID     *int64          `json:"signal_id"`
			SignalTitle  *string         `json:"signal_title"`
			DraftType    string          `json:"draft_type"`
			Subject      string          `json:"subject"`
			Body         string          `json:"body"`
			ContextUsed  json.RawMessage `json:"context_used"`
			Status       string          `json:"status"`
			CostUSD      float64         `json:"cost_usd"`
			CreatedAt    string          `json:"created_at"`
		}
		var result []draft
		for rows.Next() {
			var d draft
			var ctxJSON string
			rows.Scan(&d.ID, &d.SignalID, &d.SignalTitle,
				&d.DraftType, &d.Subject, &d.Body, &ctxJSON,
				&d.Status, &d.CostUSD, &d.CreatedAt)
			d.ContextUsed = json.RawMessage(ctxJSON)
			result = append(result, d)
		}
		if result == nil {
			result = []draft{}
		}
		writeJSON(w, http.StatusOK, result)
	})
}

// PATCH /api/v1/agent-drafts/{id} — update status
func patchDraftHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			writeErr(w, http.StatusMethodNotAllowed, "PATCH only")
			return
		}
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid id")
			return
		}

		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		allowed := map[string]bool{"draft": true, "approved": true, "discarded": true}
		if !allowed[body.Status] {
			writeErr(w, http.StatusBadRequest, "status must be draft|approved|discarded")
			return
		}

		_, err = db.ExecContext(r.Context(),
			`UPDATE agent_drafts SET status=$1, reviewed_at=NOW() WHERE id=$2`,
			body.Status, id)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": body.Status})
	})
}

// POST /api/v1/agent-drafts/generate — trigger the tech-transfer agent
func generateDraftHandler(db *sql.DB, router *llm.Router, logger *llm.DBLogger) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}

		var req struct {
			SignalID  *int64 `json:"signal_id"`
			Goal      string `json:"goal"` // free-form goal when no signal_id
			DraftType string `json:"draft_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		// Check daily budget before running agent (which is expensive)
		if err := llm.CheckBudget(r.Context(), db, router.DailyCostLimit()); err != nil {
			writeErr(w, http.StatusPaymentRequired, err.Error())
			return
		}

		ctx := r.Context()
		goal := req.Goal

		// Load signal context if a signal_id is given
		var signalID *int64
		if req.SignalID != nil {
			var (
				sTitle, sDesc string
				sScore        float64
			)
			err := db.QueryRowContext(ctx,
				`SELECT title, COALESCE(description,''), score FROM signals WHERE id=$1`,
				*req.SignalID,
			).Scan(&sTitle, &sDesc, &sScore)
			if err != nil {
				if err == sql.ErrNoRows {
					writeErr(w, http.StatusNotFound, "signal not found")
					return
				}
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			signalID = req.SignalID
			goal = fmt.Sprintf("Sinal de match: %s\n\nDescrição: %s\n\nScore: %.2f\n\n%s",
				sTitle, sDesc, sScore, req.Goal)
		}

		if goal == "" {
			writeErr(w, http.StatusBadRequest, "signal_id or goal is required")
			return
		}

		draftType := req.DraftType
		if draftType == "" {
			draftType = "email_intro"
		}

		// Build and run the agent
		braveKey := router.BraveAPIKey()
		tools := agents.DefaultTools(db, braveKey)
		agent := &agents.Agent{
			Name:     "tech_transfer",
			System:   agents.TechTransferSystemPrompt,
			Tools:    tools,
			LLM:      router,
			MaxSteps: 6,
		}

		agentResult, err := agent.Run(ctx, goal)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "agent failed: "+err.Error())
			return
		}

		// Parse agent output as {subject, body}
		var emailDraft struct {
			Subject string `json:"subject"`
			Body    string `json:"body"`
		}
		if jsonErr := json.Unmarshal([]byte(agentResult.Text), &emailDraft); jsonErr != nil {
			// Fallback: treat entire text as body
			emailDraft.Subject = "Proposta de colaboração — UFV"
			emailDraft.Body = agentResult.Text
		}

		// Calculate total cost from context log
		contextJSON, _ := json.Marshal(map[string]any{
			"steps_used":  agentResult.StepsUsed,
			"context_log": agentResult.ContextLog,
		})

		var draftID int64
		err = db.QueryRowContext(ctx, `
			INSERT INTO agent_drafts
			  (signal_id, draft_type, subject, body, context_used)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			signalID, draftType, emailDraft.Subject, emailDraft.Body, string(contextJSON),
		).Scan(&draftID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to save draft: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":          draftID,
			"subject":     emailDraft.Subject,
			"body":        emailDraft.Body,
			"steps_used":  agentResult.StepsUsed,
			"context_log": agentResult.ContextLog,
		})
	})
}
