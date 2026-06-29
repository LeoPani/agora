package agents

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Tool is the interface that every agent tool must implement.
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// ── Researcher Profile ─────────────────────────────────────────────────────

type ResearcherTool struct{ DB *sql.DB }

func (t *ResearcherTool) Name() string { return "get_researcher_profile" }
func (t *ResearcherTool) Description() string {
	return "Busca o perfil completo de um pesquisador da UFV: publicações recentes, área de pesquisa, co-autorias."
}
func (t *ResearcherTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "description": "Nome do pesquisador"},
		},
		"required": []string{"name"},
	}
}
func (t *ResearcherTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return `{"error":"name is required"}`, nil
	}

	type row struct {
		ID         int64   `json:"id"`
		Name       string  `json:"name"`
		OrcidID    *string `json:"orcid_id,omitempty"`
		Department *string `json:"department,omitempty"`
		PubCount   int     `json:"publication_count"`
	}
	var r row
	err := t.DB.QueryRowContext(ctx, `
		SELECT r.id, r.name, r.orcid_id, r.department,
		       COUNT(p.id) as pub_count
		FROM researchers r
		LEFT JOIN publications p ON p.source = 'openalex'
		  AND p.title ILIKE '%' || $1 || '%'
		WHERE r.name ILIKE $2
		GROUP BY r.id, r.name, r.orcid_id, r.department
		LIMIT 1`, name, "%"+name+"%",
	).Scan(&r.ID, &r.Name, &r.OrcidID, &r.Department, &r.PubCount)
	if err == sql.ErrNoRows {
		return fmt.Sprintf(`{"error":"researcher %q not found"}`, name), nil
	}
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(r)
	return string(b), nil
}

// ── Publication Search ─────────────────────────────────────────────────────

type PublicationSearchTool struct{ DB *sql.DB }

func (t *PublicationSearchTool) Name() string { return "search_publications" }
func (t *PublicationSearchTool) Description() string {
	return "Busca publicações da UFV por palavras-chave no título ou abstract."
}
func (t *PublicationSearchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Termos de busca"},
			"limit": map[string]any{"type": "integer", "description": "Máximo de resultados (default 5)"},
		},
		"required": []string{"query"},
	}
}
func (t *PublicationSearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	rows, err := t.DB.QueryContext(ctx, `
		SELECT id, title, COALESCE(abstract,'') as abstract,
		       publication_year, cited_by_count
		FROM publications
		WHERE title ILIKE $1 OR abstract ILIKE $1
		ORDER BY cited_by_count DESC
		LIMIT $2`, "%"+query+"%", limit)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type pub struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Year  *int   `json:"year,omitempty"`
		Cited int    `json:"cited_by_count"`
	}
	var pubs []pub
	for rows.Next() {
		var p pub
		var abs string
		rows.Scan(&p.ID, &p.Title, &abs, &p.Year, &p.Cited)
		pubs = append(pubs, p)
	}
	if len(pubs) == 0 {
		return `{"results":[],"message":"Nenhuma publicação encontrada"}`, nil
	}
	b, _ := json.Marshal(map[string]any{"results": pubs})
	return string(b), nil
}

// ── Patent Search ──────────────────────────────────────────────────────────

type PatentSearchTool struct{ DB *sql.DB }

func (t *PatentSearchTool) Name() string { return "search_patents" }
func (t *PatentSearchTool) Description() string {
	return "Busca patentes por palavras-chave no título ou abstract."
}
func (t *PatentSearchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
			"ufv_only": map[string]any{
				"type":        "boolean",
				"description": "Filtrar somente patentes UFV",
			},
		},
		"required": []string{"query"},
	}
}
func (t *PatentSearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	ufvOnly, _ := args["ufv_only"].(bool)

	q := `SELECT id, COALESCE(title,'') as title, COALESCE(abstract,'') as abstract,
	             filing_date, legal_status, applicant_type
		  FROM patents
		  WHERE (title ILIKE $1 OR abstract ILIKE $1)`
	if ufvOnly {
		q += " AND applicant_type = 'UFV'"
	}
	q += " LIMIT 5"

	rows, err := t.DB.QueryContext(ctx, q, "%"+query+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type pat struct {
		ID       int64   `json:"id"`
		Title    string  `json:"title"`
		Date     *string `json:"filing_date,omitempty"`
		Status   *string `json:"status,omitempty"`
		Applicant *string `json:"applicant_type,omitempty"`
	}
	var pats []pat
	for rows.Next() {
		var p pat
		var abs string
		rows.Scan(&p.ID, &p.Title, &abs, &p.Date, &p.Status, &p.Applicant)
		pats = append(pats, p)
	}
	if len(pats) == 0 {
		return `{"results":[],"message":"Nenhuma patente encontrada"}`, nil
	}
	b, _ := json.Marshal(map[string]any{"results": pats})
	return string(b), nil
}

// ── Web Search (Brave API — optional) ─────────────────────────────────────

type WebSearchTool struct {
	APIKey string
	HTTP   *http.Client
}

func (t *WebSearchTool) Name() string { return "web_search" }
func (t *WebSearchTool) Description() string {
	return "Busca notícias e informações recentes sobre uma empresa ou tecnologia na web."
}
func (t *WebSearchTool) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Termos de busca"},
		},
		"required": []string{"query"},
	}
}
func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.APIKey == "" {
		return `{"error":"BRAVE_API_KEY não configurado — busca web desabilitada"}`, nil
	}
	query, _ := args["query"].(string)

	hc := t.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	req, _ := http.NewRequestWithContext(ctx, "GET",
		"https://api.search.brave.com/res/v1/web/search?q="+strings.ReplaceAll(query, " ", "+")+"&count=5",
		nil,
	)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", t.APIKey)

	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Sprintf(`{"error":"web search failed: %s"}`, err.Error()), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32_000))

	// Extract only the useful snippets from Brave response
	var br braveResponse
	if err := json.Unmarshal(body, &br); err != nil {
		return `{"error":"failed to parse search results"}`, nil
	}

	type result struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	}
	var results []result
	for _, r := range br.Web.Results {
		results = append(results, result{r.Title, r.URL, r.Description})
	}
	out, _ := json.Marshal(map[string]any{"results": results})
	return string(out), nil
}

type braveResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}

// ── Tech Transfer Agent factory ────────────────────────────────────────────

const TechTransferSystemPrompt = `Você é um assistente especializado do NIT (Núcleo de Inovação Tecnológica) da UFV.
Quando recebe um sinal de match entre um pesquisador e uma empresa, você:
1. Investiga o contexto usando as ferramentas disponíveis
2. Redige um email de aproximação profissional e personalizado em português formal

REGRAS DO EMAIL:
- Máximo 200 palavras
- Cite UM ponto específico relevante do pesquisador ou sua pesquisa
- Cite UMA razão específica pela qual a empresa pode ter interesse
- Proponha um próximo passo concreto (ex: "uma conversa de 30 minutos")
- Tom: cordial, profissional, sem ser excessivamente formal

Quando tiver informação suficiente, gere APENAS um JSON com:
{"subject": "Assunto do email", "body": "Corpo completo do email"}`

// DefaultTools returns the standard toolset for a tech-transfer agent.
func DefaultTools(db *sql.DB, braveAPIKey string) []Tool {
	hc := &http.Client{Timeout: 10 * time.Second}
	return []Tool{
		&ResearcherTool{DB: db},
		&PublicationSearchTool{DB: db},
		&PatentSearchTool{DB: db},
		&WebSearchTool{APIKey: braveAPIKey, HTTP: hc},
	}
}

// Ensure bytes package is used
var _ = bytes.NewReader
