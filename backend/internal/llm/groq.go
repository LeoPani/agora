package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const groqBaseURL = "https://api.groq.com/openai/v1/chat/completions"

type GroqClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewGroqClient(apiKey, model string, hc *http.Client) *GroqClient {
	return &GroqClient{apiKey: apiKey, model: model, http: hc}
}

func (g *GroqClient) ProviderName() Provider { return ProviderGroq }
func (g *GroqClient) ModelName() string      { return g.model }

func (g *GroqClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	msgs := buildMessages(req)

	body := map[string]any{
		"model":       g.model,
		"messages":    msgs,
		"temperature": req.Temperature,
		"max_tokens":  orDefault(req.MaxTokens, 4096),
	}
	if req.JSONMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	if len(req.Tools) > 0 {
		body["tools"] = groqTools(req.Tools)
		body["tool_choice"] = "auto"
	}

	rawBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", groqBaseURL, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gr groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("groq: decode error: %w", err)
	}
	if gr.Error != nil {
		return nil, fmt.Errorf("groq: %s", gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return nil, fmt.Errorf("groq: empty choices")
	}
	choice := gr.Choices[0]

	out := &CompletionResponse{
		Text:         choice.Message.Content,
		PromptTokens: gr.Usage.PromptTokens,
		OutputTokens: gr.Usage.CompletionTokens,
	}
	// tool calls
	for _, tc := range choice.Message.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:       tc.ID,
			Name:     tc.Function.Name,
			ArgsJSON: tc.Function.Arguments,
		})
	}
	out.CostUSD = CalcCost(ProviderGroq, g.model, out.PromptTokens, out.OutputTokens)
	return out, nil
}

// ── wire types ──────────────────────────────────────────────────────────────

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func groqTools(tools []Tool) []map[string]any {
	out := make([]map[string]any, len(tools))
	for i, t := range tools {
		out[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		}
	}
	return out
}

func buildMessages(req CompletionRequest) []map[string]any {
	if len(req.Messages) > 0 {
		out := make([]map[string]any, 0, len(req.Messages))
		for _, m := range req.Messages {
			msg := map[string]any{"role": m.Role, "content": m.Content}
			if m.ToolCallID != "" {
				msg["tool_call_id"] = m.ToolCallID
			}
			if len(m.ToolCalls) > 0 {
				tcs := make([]map[string]any, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					tcs[i] = map[string]any{
						"id":   tc.ID,
						"type": "function",
						"function": map[string]any{
							"name":      tc.Name,
							"arguments": tc.ArgsJSON,
						},
					}
				}
				msg["tool_calls"] = tcs
			}
			out = append(out, msg)
		}
		return out
	}
	return []map[string]any{{"role": "user", "content": req.Prompt}}
}

func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
