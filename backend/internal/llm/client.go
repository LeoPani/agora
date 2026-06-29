package llm

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Provider string

const (
	ProviderGroq      Provider = "groq"
	ProviderGemini    Provider = "gemini"
	ProviderAnthropic Provider = "anthropic"
	ProviderOllama    Provider = "ollama"
)

// Message represents a single turn in a conversation.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall is a function invocation requested by the LLM.
type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ArgsJSON string `json:"arguments"`
}

// Tool describes a function the LLM can invoke.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type CompletionRequest struct {
	Purpose     string
	Prompt      string    // used when Messages is nil (single-turn)
	Messages    []Message // multi-turn; overrides Prompt
	Tools       []Tool
	Temperature float64
	MaxTokens   int
	JSONMode    bool
	Provider    Provider // force a provider; empty = auto-route
	Metadata    map[string]any
}

type CompletionResponse struct {
	Text         string
	ToolCalls    []ToolCall
	PromptTokens int
	OutputTokens int
	CostUSD      float64
	LatencyMs    int
	Provider     Provider
	Model        string
}

// LLMClient is the interface every provider implements.
type LLMClient interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	ProviderName() Provider
	ModelName() string
}

// Config holds LLM credentials and defaults loaded from environment.
type Config struct {
	GroqAPIKey      string
	GeminiAPIKey    string
	AnthropicAPIKey string
	BraveAPIKey     string
	OllamaHost      string

	DefaultProvider Provider
	DefaultModel    string
	HeavyModel      string

	EnableLogging  bool
	LogPath        string
	DailyCostLimit float64
}

func LoadConfig() Config {
	return Config{
		GroqAPIKey:      os.Getenv("GROQ_API_KEY"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		BraveAPIKey:     os.Getenv("BRAVE_API_KEY"),
		OllamaHost:      envOr("OLLAMA_HOST", "http://localhost:11434"),

		DefaultProvider: Provider(envOr("LLM_PROVIDER_DEFAULT", "groq")),
		DefaultModel:    envOr("LLM_MODEL_DEFAULT", "llama-3.3-70b-versatile"),
		HeavyModel:      envOr("LLM_MODEL_HEAVY", "gemini-2.0-flash"),

		EnableLogging:  os.Getenv("ENABLE_LLM_LOGGING") != "false",
		LogPath:        envOr("LLM_LOG_PATH", "./logs/llm_calls.jsonl"),
		DailyCostLimit: parseFloat64(os.Getenv("LLM_COST_LIMIT_DAILY_USD"), 5.00),
	}
}

// Router dispatches completion requests to the right provider.
type Router struct {
	cfg     Config
	clients map[Provider]LLMClient
	http    *http.Client
}

func NewRouter(cfg Config) *Router {
	r := &Router{
		cfg:     cfg,
		clients: make(map[Provider]LLMClient),
		http:    &http.Client{Timeout: 120 * time.Second},
	}
	if cfg.GroqAPIKey != "" {
		r.clients[ProviderGroq] = NewGroqClient(cfg.GroqAPIKey, cfg.DefaultModel, r.http)
	}
	if cfg.GeminiAPIKey != "" {
		r.clients[ProviderGemini] = NewGeminiClient(cfg.GeminiAPIKey, cfg.HeavyModel, r.http)
	}
	if cfg.AnthropicAPIKey != "" {
		r.clients[ProviderAnthropic] = NewAnthropicClient(cfg.AnthropicAPIKey, "claude-sonnet-4-6", r.http)
	}
	r.clients[ProviderOllama] = NewOllamaClient(cfg.OllamaHost, "llama3.1:8b", r.http)
	return r
}

// BraveAPIKey exposes the key for use by the web-search tool.
func (r *Router) BraveAPIKey() string { return r.cfg.BraveAPIKey }

func (r *Router) DailyCostLimit() float64 { return r.cfg.DailyCostLimit }

func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	provider := chooseProvider(req.Purpose, req.Provider, r.cfg)
	client, ok := r.clients[provider]
	if !ok {
		// fallback chain
		for _, p := range []Provider{ProviderGroq, ProviderGemini, ProviderAnthropic, ProviderOllama} {
			if c, exists := r.clients[p]; exists {
				client = c
				provider = p
				break
			}
		}
	}
	if client == nil {
		return nil, errorf("llm: no provider configured (tried %q)", provider)
	}

	start := time.Now()
	resp, err := client.Complete(ctx, req)
	if resp != nil {
		resp.LatencyMs = int(time.Since(start).Milliseconds())
		resp.Provider = client.ProviderName()
		resp.Model = client.ModelName()
	}
	return resp, err
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseFloat64(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}
