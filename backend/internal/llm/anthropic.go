package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const anthropicBaseURL = "https://api.anthropic.com/v1/messages"

type AnthropicClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewAnthropicClient(apiKey, model string, hc *http.Client) *AnthropicClient {
	return &AnthropicClient{apiKey: apiKey, model: model, http: hc}
}

func (a *AnthropicClient) ProviderName() Provider { return ProviderAnthropic }
func (a *AnthropicClient) ModelName() string      { return a.model }

func (a *AnthropicClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	type antMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	var msgs []antMsg
	if len(req.Messages) > 0 {
		for _, m := range req.Messages {
			if m.Role == "system" {
				continue // handled separately
			}
			msgs = append(msgs, antMsg{Role: m.Role, Content: m.Content})
		}
	} else {
		msgs = []antMsg{{Role: "user", Content: req.Prompt}}
	}

	body := map[string]any{
		"model":      a.model,
		"max_tokens": orDefault(req.MaxTokens, 4096),
		"messages":   msgs,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.JSONMode {
		// Anthropic doesn't have a strict JSON mode — instruct via prompt
		body["system"] = "Responda APENAS com JSON válido, sem markdown, sem texto antes ou depois."
	}

	rawBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicBaseURL, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("anthropic: decode error: %w", err)
	}
	if ar.Error != nil {
		return nil, fmt.Errorf("anthropic: %s: %s", ar.Error.Type, ar.Error.Message)
	}
	if len(ar.Content) == 0 {
		return nil, fmt.Errorf("anthropic: empty content")
	}

	pt := ar.Usage.InputTokens
	ct := ar.Usage.OutputTokens
	return &CompletionResponse{
		Text:         ar.Content[0].Text,
		PromptTokens: pt,
		OutputTokens: ct,
		CostUSD:      CalcCost(ProviderAnthropic, a.model, pt, ct),
	}, nil
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
