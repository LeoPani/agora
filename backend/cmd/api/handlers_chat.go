package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/LeoPani/agora/backend/internal/llm"
	"github.com/LeoPani/agora/backend/internal/rag"
)

const ragSystemPrompt = `Você é o Oráculo do Ágora, assistente especializado em inteligência de inovação universitária da UFV.

Responda às perguntas usando APENAS as fontes fornecidas no contexto abaixo.

REGRAS OBRIGATÓRIAS:
- Cite as fontes com [1], [2], etc.
- Se a resposta não estiver no contexto, diga "Não encontrei essa informação nos dados disponíveis."
- Seja conciso. Responda em português brasileiro.
- Quando listar pesquisadores ou patentes, use formato de lista.`

// POST /api/chat
func chatHandler(db *sql.DB, retriever *rag.Retriever, router *llm.Router, logger *llm.DBLogger) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}

		var req struct {
			ConversationID string `json:"conversation_id"`
			Message        string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.Message) == "" {
			writeErr(w, http.StatusBadRequest, "message is required")
			return
		}

		ctx := r.Context()

		// Create or retrieve conversation
		convID := req.ConversationID
		if convID == "" {
			title := req.Message
			if len(title) > 80 {
				title = title[:80] + "…"
			}
			err := db.QueryRowContext(ctx,
				`INSERT INTO conversations (title) VALUES ($1) RETURNING id::text`, title,
			).Scan(&convID)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "failed to create conversation")
				return
			}
		}

		// Save user message
		db.ExecContext(ctx,
			`INSERT INTO conversation_messages (conversation_id, role, content) VALUES ($1,'user',$2)`,
			convID, req.Message,
		)

		// Retrieve relevant chunks
		chunks, err := retriever.Search(ctx, req.Message, 10)
		if err != nil {
			chunks = nil
		}

		// Build context string
		var ctxParts []string
		for i, c := range chunks {
			label := sourceLabel(c.SourceType)
			snippet := c.Content
			if len(snippet) > 400 {
				snippet = snippet[:400] + "…"
			}
			ctxParts = append(ctxParts, fmt.Sprintf("[%d] %s \"%s\": %s", i+1, label, c.Title, snippet))
		}

		var prompt string
		if len(ctxParts) > 0 {
			prompt = ragSystemPrompt + "\n\nCONTEXTO:\n" + strings.Join(ctxParts, "\n\n") +
				"\n\nPERGUNTA: " + req.Message + "\n\nRESPOSTA:"
		} else {
			prompt = ragSystemPrompt + "\n\n[Nenhum contexto encontrado no banco de dados]\n\nPERGUNTA: " +
				req.Message + "\n\nRESPOSTA:"
		}

		// Build sources array (before LLM call so fallback can use it)
		type sourceRef struct {
			Index      int    `json:"index"`
			SourceType string `json:"source_type"`
			ID         int64  `json:"id"`
			Title      string `json:"title"`
			URL        string `json:"url,omitempty"`
		}
		var sources []sourceRef
		for i, c := range chunks {
			src := sourceRef{Index: i + 1, SourceType: c.SourceType, ID: c.ID, Title: c.Title}
			if c.URL != nil {
				src.URL = *c.URL
			}
			sources = append(sources, src)
		}
		if sources == nil {
			sources = []sourceRef{}
		}

		// Call LLM
		completionReq := llm.CompletionRequest{
			Purpose:     "rag_query",
			Prompt:      prompt,
			Temperature: 0.3,
			MaxTokens:   1024,
		}
		resp, llmErr := router.Complete(ctx, completionReq)

		var llmCallID int64
		if logger != nil {
			llmCallID = logger.Log(ctx, completionReq, resp, llmErr)
		}

		// Fallback: sem LLM, monta resposta formatada a partir dos chunks
		var answerText string
		if llmErr != nil {
			if len(chunks) == 0 {
				answerText = "Nenhum resultado encontrado para sua busca. " +
					"_(Modo sem LLM — configure GROQ_API_KEY para respostas em linguagem natural.)_"
			} else {
				var lines []string
				lines = append(lines, fmt.Sprintf("Encontrei **%d resultado(s)** para \"_%s_\":\n", len(chunks), req.Message))
				for i, c := range chunks {
					snippet := c.Content
					if len(snippet) > 200 {
						snippet = snippet[:200] + "…"
					}
					lines = append(lines, fmt.Sprintf("**[%d] %s** — %s\n%s", i+1, sourceLabel(c.SourceType), c.Title, snippet))
				}
				lines = append(lines, "\n_(Modo sem LLM — configure GROQ_API_KEY para síntese em linguagem natural.)_")
				answerText = strings.Join(lines, "\n\n")
			}
		} else {
			answerText = resp.Text
		}

		sourcesJSON, _ := json.Marshal(sources)

		// Save assistant message
		db.ExecContext(ctx, `
			INSERT INTO conversation_messages
			  (conversation_id, role, content, sources, llm_call_id)
			VALUES ($1, 'assistant', $2, $3, $4)`,
			convID, answerText, string(sourcesJSON), nullInt64(llmCallID),
		)

		// Update conversation timestamp
		db.ExecContext(ctx,
			`UPDATE conversations SET updated_at=NOW() WHERE id=$1::uuid`, convID)

		costUSD := 0.0
		if resp != nil {
			costUSD = resp.CostUSD
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"conversation_id": convID,
			"message":         answerText,
			"sources":         sources,
			"cost_usd":        costUSD,
			"llm_available":   llmErr == nil,
		})
	})
}

// GET /api/conversations
func conversationsHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `
			SELECT id::text, COALESCE(title,'Nova conversa'), created_at, updated_at
			FROM conversations ORDER BY updated_at DESC LIMIT 50`)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type conv struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}
		var result []conv
		for rows.Next() {
			var c conv
			rows.Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
			result = append(result, c)
		}
		if result == nil {
			result = []conv{}
		}
		writeJSON(w, http.StatusOK, result)
	})
}

// GET /api/conversations/{id}/messages
func conversationMessagesHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeErr(w, http.StatusBadRequest, "id required")
			return
		}

		rows, err := db.QueryContext(r.Context(), `
			SELECT id, role, content, COALESCE(sources::text,'[]'), created_at
			FROM conversation_messages
			WHERE conversation_id=$1::uuid
			ORDER BY created_at`, id)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type msg struct {
			ID        int64           `json:"id"`
			Role      string          `json:"role"`
			Content   string          `json:"content"`
			Sources   json.RawMessage `json:"sources"`
			CreatedAt string          `json:"created_at"`
		}
		var result []msg
		for rows.Next() {
			var m msg
			var srcStr string
			rows.Scan(&m.ID, &m.Role, &m.Content, &srcStr, &m.CreatedAt)
			m.Sources = json.RawMessage(srcStr)
			result = append(result, m)
		}
		if result == nil {
			result = []msg{}
		}
		writeJSON(w, http.StatusOK, result)
	})
}

func sourceLabel(t string) string {
	switch t {
	case "publication":
		return "Publicação"
	case "patent":
		return "Patente"
	case "opportunity":
		return "Edital"
	}
	return t
}

func nullInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
