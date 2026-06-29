package llm

// chooseProvider selects which provider to use based on task purpose and any
// explicit override in the request.
func chooseProvider(purpose string, override Provider, cfg Config) Provider {
	if override != "" {
		return override
	}
	switch purpose {
	case "extract_edital", "classify_simple", "rag_query":
		// Rápido e barato → Groq
		if cfg.GroqAPIKey != "" {
			return ProviderGroq
		}
	case "agent_action", "complex_reasoning":
		// Raciocínio complexo → Gemini ou Claude
		if cfg.GeminiAPIKey != "" {
			return ProviderGemini
		}
		if cfg.AnthropicAPIKey != "" {
			return ProviderAnthropic
		}
	case "batch_classification":
		// Volume gigante → Ollama local (custo zero)
		return ProviderOllama
	}
	// Padrão
	return cfg.DefaultProvider
}
