package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type GeminiClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewGeminiClient(apiKey, model string, hc *http.Client) *GeminiClient {
	return &GeminiClient{apiKey: apiKey, model: model, http: hc}
}

func (g *GeminiClient) ProviderName() Provider { return ProviderGemini }
func (g *GeminiClient) ModelName() string      { return g.model }

func (g *GeminiClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		g.model, g.apiKey,
	)

	var parts []map[string]any
	if len(req.Messages) > 0 {
		last := req.Messages[len(req.Messages)-1]
		parts = []map[string]any{{"text": last.Content}}
	} else {
		parts = []map[string]any{{"text": req.Prompt}}
	}

	genCfg := map[string]any{
		"temperature":     req.Temperature,
		"maxOutputTokens": orDefault(req.MaxTokens, 4096),
	}
	if req.JSONMode {
		genCfg["responseMimeType"] = "application/json"
	}

	body := map[string]any{
		"contents":         []map[string]any{{"parts": parts}},
		"generationConfig": genCfg,
	}

	rawBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gr geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("gemini: decode error: %w", err)
	}
	if gr.Error != nil {
		return nil, fmt.Errorf("gemini: %s", gr.Error.Message)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini: empty candidates")
	}

	text := gr.Candidates[0].Content.Parts[0].Text
	pt := gr.UsageMetadata.PromptTokenCount
	ct := gr.UsageMetadata.CandidatesTokenCount

	return &CompletionResponse{
		Text:         text,
		PromptTokens: pt,
		OutputTokens: ct,
		CostUSD:      CalcCost(ProviderGemini, g.model, pt, ct),
	}, nil
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}
