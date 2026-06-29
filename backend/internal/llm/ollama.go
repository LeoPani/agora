package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type OllamaClient struct {
	host  string
	model string
	http  *http.Client
}

func NewOllamaClient(host, model string, hc *http.Client) *OllamaClient {
	return &OllamaClient{host: host, model: model, http: hc}
}

func (o *OllamaClient) ProviderName() Provider { return ProviderOllama }
func (o *OllamaClient) ModelName() string      { return o.model }

func (o *OllamaClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	msgs := buildMessages(req)
	body := map[string]any{
		"model":    o.model,
		"messages": msgs,
		"stream":   false,
		"options": map[string]any{
			"temperature": req.Temperature,
			"num_predict": orDefault(req.MaxTokens, 4096),
		},
	}
	if req.JSONMode {
		body["format"] = "json"
	}

	rawBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.host+"/api/chat", bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: %w (is 'ollama serve' running?)", err)
	}
	defer resp.Body.Close()

	var or_ ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&or_); err != nil {
		return nil, fmt.Errorf("ollama: decode error: %w", err)
	}

	pt := or_.PromptEvalCount
	ct := or_.EvalCount
	return &CompletionResponse{
		Text:         or_.Message.Content,
		PromptTokens: pt,
		OutputTokens: ct,
		CostUSD:      0, // local = free
	}, nil
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	PromptEvalCount int `json:"prompt_eval_count"`
	EvalCount       int `json:"eval_count"`
}
