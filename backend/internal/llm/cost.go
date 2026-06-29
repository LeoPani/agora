package llm

import (
	"context"
	"database/sql"
	"fmt"
)

// Preços aproximados por 1M tokens (USD), set 2025.
type priceEntry struct{ input, output float64 }

var priceTable = map[Provider]map[string]priceEntry{
	ProviderGroq: {
		"llama-3.3-70b-versatile":   {0.59, 0.79},
		"llama-3.1-70b-versatile":   {0.59, 0.79},
		"llama-3.1-8b-instant":      {0.05, 0.08},
		"llama3-8b-8192":            {0.05, 0.08},
		"mixtral-8x7b-32768":        {0.24, 0.24},
	},
	ProviderGemini: {
		"gemini-2.0-flash":          {0.075, 0.30},
		"gemini-1.5-flash":          {0.075, 0.30},
		"gemini-1.5-pro":            {1.25, 5.00},
	},
	ProviderAnthropic: {
		"claude-sonnet-4-6":         {3.00, 15.00},
		"claude-haiku-4-5-20251001": {0.80, 4.00},
	},
	ProviderOllama: {},
}

// CalcCost returns estimated cost in USD for a completion.
func CalcCost(provider Provider, model string, promptTokens, outputTokens int) float64 {
	if models, ok := priceTable[provider]; ok {
		if p, ok := models[model]; ok {
			return (float64(promptTokens)*p.input + float64(outputTokens)*p.output) / 1_000_000
		}
	}
	return 0
}

// CheckBudget returns an error if the daily spend has reached the limit.
func CheckBudget(ctx context.Context, db *sql.DB, limit float64) error {
	var total float64
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM llm_calls
		WHERE created_at >= CURRENT_DATE AND success = TRUE
	`).Scan(&total)
	if err != nil {
		return nil // don't block calls on a budget query failure
	}
	if total >= limit {
		return fmt.Errorf("llm: daily budget exceeded (%.4f >= %.2f USD)", total, limit)
	}
	return nil
}
