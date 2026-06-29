package llm

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// DBLogger persists LLM call records to the llm_calls table and optionally
// to a JSONL file for offline analysis.
type DBLogger struct {
	db      *sql.DB
	logPath string
	mu      sync.Mutex
	file    *os.File
}

func NewDBLogger(db *sql.DB, logPath string) *DBLogger {
	l := &DBLogger{db: db, logPath: logPath}
	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err == nil {
			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err == nil {
				l.file = f
			}
		}
	}
	return l
}

type logEntry struct {
	Purpose   string         `json:"purpose"`
	Provider  string         `json:"provider"`
	Model     string         `json:"model"`
	PromptTok int            `json:"prompt_tokens"`
	OutTok    int            `json:"completion_tokens"`
	TotalTok  int            `json:"total_tokens"`
	CostUSD   float64        `json:"cost_usd"`
	LatencyMs int            `json:"latency_ms"`
	Success   bool           `json:"success"`
	ErrMsg    string         `json:"error_message,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Log writes the call to the DB and optional JSONL file, returning the row ID.
func (l *DBLogger) Log(ctx context.Context, req CompletionRequest, resp *CompletionResponse, callErr error) int64 {
	var (
		provider, model string
		prompt, out     int
		costUSD         float64
		latencyMs       int
		success         = callErr == nil
		errMsg          string
	)
	if resp != nil {
		provider  = string(resp.Provider)
		model     = resp.Model
		prompt    = resp.PromptTokens
		out       = resp.OutputTokens
		costUSD   = resp.CostUSD
		latencyMs = resp.LatencyMs
	}
	if callErr != nil {
		errMsg = callErr.Error()
	}
	total := prompt + out

	var metaJSON []byte
	if req.Metadata != nil {
		metaJSON, _ = json.Marshal(req.Metadata)
	}

	var id int64
	err := l.db.QueryRowContext(ctx, `
		INSERT INTO llm_calls
		  (purpose, provider, model, prompt_tokens, completion_tokens, total_tokens,
		   cost_usd, latency_ms, success, error_message, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`,
		req.Purpose, provider, model, prompt, out, total,
		costUSD, latencyMs, success, nullableStr(errMsg), nullableJSON(metaJSON),
	).Scan(&id)
	if err != nil {
		slog.Warn("llm_calls insert failed", "err", err)
	}

	if l.file != nil {
		entry := logEntry{
			Purpose:   req.Purpose,
			Provider:  provider,
			Model:     model,
			PromptTok: prompt,
			OutTok:    out,
			TotalTok:  total,
			CostUSD:   costUSD,
			LatencyMs: latencyMs,
			Success:   success,
			ErrMsg:    errMsg,
			Metadata:  req.Metadata,
		}
		raw, _ := json.Marshal(entry)
		l.mu.Lock()
		l.file.Write(raw)
		l.file.WriteString("\n")
		l.mu.Unlock()
	}
	return id
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}
