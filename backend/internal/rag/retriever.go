package rag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
)

// Chunk is a retrieved piece of content from the data lake.
type Chunk struct {
	ID         int64
	SourceType string // "publication" | "patent" | "opportunity"
	Title      string
	Content    string // abstract or description
	URL        *string
	Score      float64
}

// Retriever does hybrid (vector + full-text) search with Reciprocal Rank Fusion.
type Retriever struct {
	db       *sql.DB
	embedURL string
	http     *http.Client
}

func New(db *sql.DB, embedServerURL string) *Retriever {
	if embedServerURL == "" {
		embedServerURL = "http://localhost:8082"
	}
	return &Retriever{
		db:       db,
		embedURL: embedServerURL,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Search returns the top-k most relevant chunks for the query.
func (r *Retriever) Search(ctx context.Context, query string, k int) ([]Chunk, error) {
	groupResults := r.researchGroupSearch(ctx, query, k)
	fundingResults := r.fundingSearch(ctx, query, k)

	emb, err := r.embed(ctx, query)
	var (
		vecResults  []Chunk
		textResults []Chunk
	)
	if err == nil {
		vecResults = r.vectorSearch(ctx, emb, k*2)
	}
	textResults = r.textSearch(ctx, query, k*2)

	merged := rrf(k, groupResults, vecResults, textResults)

	// Injeta editais nos primeiros slots quando a query é sobre financiamento.
	// Reserva até 4 das 10 posições para oportunidades, o resto para publicações/grupos.
	if len(fundingResults) > 0 {
		seen := map[string]bool{}
		for _, c := range merged {
			seen[fmt.Sprintf("%s:%d", c.SourceType, c.ID)] = true
		}
		const maxFunding = 4
		var out []Chunk
		fi := 0
		for _, c := range fundingResults {
			if fi >= maxFunding {
				break
			}
			key := fmt.Sprintf("%s:%d", c.SourceType, c.ID)
			if !seen[key] {
				out = append(out, c)
				seen[key] = true
				fi++
			}
		}
		for _, c := range merged {
			if len(out) >= k {
				break
			}
			out = append(out, c)
		}
		return out, nil
	}

	return merged, nil
}

// fundingSearch busca oportunidades/editais quando a query é sobre financiamento.
// Usa ILIKE por keywords (mais robusto que FTS para títulos curtos de editais).
func (r *Retriever) fundingSearch(ctx context.Context, query string, limit int) []Chunk {
	qLow := strings.ToLower(query)
	fundingSignals := []string{
		"edital", "chamada", "bolsa", "financiamento", "fomento",
		"apoio", "fapemig", "finep", "cnpq", "bndes", "embrapii",
		"exist", "abertas", "aberto", "disponív", "recurso", "grant",
	}
	hasFunding := false
	for _, kw := range fundingSignals {
		if strings.Contains(qLow, kw) {
			hasFunding = true
			break
		}
	}
	if !hasFunding {
		return nil
	}

	// Extrai keywords temáticas para filtrar oportunidades relevantes
	kws := queryKeywords(query)
	var rows *sql.Rows
	var err error

	if len(kws) > 0 {
		// Busca oportunidades que mencionen alguma keyword temática (ILIKE por cada kw)
		// Usa ILIKE simples em vez de FTS pois os títulos de editais são curtos e específicos
		like := "%" + strings.Join(kws, "%") + "%"
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, title, COALESCE(description,'') AS content, url, 0.85 AS score
			FROM opportunities
			WHERE description ILIKE $1 OR title ILIKE $1
			ORDER BY collected_at DESC
			LIMIT $2`, like, limit)
	}
	// Fallback: retorna todas as oportunidades recentes
	if err != nil || rows == nil {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, title, COALESCE(description,'') AS content, url, 0.8 AS score
			FROM opportunities
			ORDER BY collected_at DESC
			LIMIT $1`, limit)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()
	var results []Chunk
	for rows.Next() {
		var c Chunk
		c.SourceType = "opportunity"
		rows.Scan(&c.ID, &c.Title, &c.Content, &c.URL, &c.Score)
		results = append(results, c)
	}
	// Se ILIKE temático retornou poucos resultados, complementa com recentes
	if len(results) < 3 && len(kws) > 0 {
		rows2, err2 := r.db.QueryContext(ctx, `
			SELECT id, title, COALESCE(description,'') AS content, url, 0.75 AS score
			FROM opportunities
			ORDER BY collected_at DESC
			LIMIT $1`, limit)
		if err2 == nil {
			defer rows2.Close()
			seen := map[int64]bool{}
			for _, c := range results {
				seen[c.ID] = true
			}
			for rows2.Next() {
				var c Chunk
				c.SourceType = "opportunity"
				rows2.Scan(&c.ID, &c.Title, &c.Content, &c.URL, &c.Score)
				if !seen[c.ID] {
					results = append(results, c)
				}
			}
		}
	}
	return results
}

// embed calls the embed_server to get a query vector.
func (r *Retriever) embed(ctx context.Context, text string) ([]float32, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		r.embedURL+"/embed?text="+urlEncode(text), nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embed server: status %d", resp.StatusCode)
	}
	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Embedding, nil
}

func (r *Retriever) vectorSearch(ctx context.Context, emb []float32, limit int) []Chunk {
	vec := pgvector.NewVector(emb)
	var results []Chunk

	// Publications (com autores UFV no início do content)
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.title,
		       COALESCE('Pesquisadores UFV: ' || (
		           SELECT string_agg(r.full_name, ', ')
		           FROM publication_authors pa
		           JOIN researchers r ON r.id = pa.researcher_id
		           WHERE pa.publication_id = p.id
		         ) || '. ', '') ||
		         COALESCE(p.abstract,'') AS content,
		       NULL::text as url,
		       1 - (p.embedding <=> $1) AS score
		FROM publications p
		WHERE p.embedding IS NOT NULL
		ORDER BY p.embedding <=> $1
		LIMIT $2`, vec, limit)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c Chunk
			c.SourceType = "publication"
			rows.Scan(&c.ID, &c.Title, &c.Content, &c.URL, &c.Score)
			results = append(results, c)
		}
	}

	// Patents
	rows2, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(title,'') as title, COALESCE(abstract,'') as content,
		       NULL::text as url, 1 - (embedding <=> $1) AS score
		FROM patents
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2`, vec, limit)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var c Chunk
			c.SourceType = "patent"
			rows2.Scan(&c.ID, &c.Title, &c.Content, &c.URL, &c.Score)
			results = append(results, c)
		}
	}

	// Opportunities
	rows3, err := r.db.QueryContext(ctx, `
		SELECT id, title, COALESCE(description,'') as content,
		       url, 1 - (embedding <=> $1) AS score
		FROM opportunities
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2`, vec, limit)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var c Chunk
			c.SourceType = "opportunity"
			rows3.Scan(&c.ID, &c.Title, &c.Content, &c.URL, &c.Score)
			results = append(results, c)
		}
	}

	return results
}

// researchGroupSearch busca grupos de pesquisa usando full-text search.
// Sempre chamado (sem dependência de embedding), pois research_groups não tem vetor ainda.
func (r *Retriever) researchGroupSearch(ctx context.Context, query string, limit int) []Chunk {
	kws := queryKeywords(query)
	if len(kws) == 0 {
		return nil
	}
	ftsQuery := strings.Join(kws, " OR ")
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,
		       name ||
		         CASE WHEN leader IS NOT NULL AND leader <> '' THEN ' (Líder: ' || leader || ')' ELSE '' END
		         AS title,
		       'Grupo de Pesquisa UFV: ' || name ||
		       ' | Departamento: ' || COALESCE(department,'') ||
		       ' | Área: ' || COALESCE(main_area,'') ||
		       ' | Linhas de pesquisa: ' || COALESCE(array_to_string(research_lines, ', '),'') AS content,
		       NULL::text AS url,
		       0.75 AS score
		FROM research_groups
		WHERE to_tsvector('portuguese',
		        COALESCE(name,'') || ' ' ||
		        COALESCE(array_to_string(research_lines,' '),'') || ' ' ||
		        COALESCE(main_area,''))
		      @@ websearch_to_tsquery('portuguese', $1)
		LIMIT $2`, ftsQuery, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var results []Chunk
	for rows.Next() {
		var c Chunk
		c.SourceType = "research_group"
		rows.Scan(&c.ID, &c.Title, &c.Content, &c.URL, &c.Score)
		results = append(results, c)
	}
	return results
}

// textSearch extrai keywords da pergunta e usa full-text search do Postgres,
// com fallback para ILIKE por palavra-chave.
func (r *Retriever) textSearch(ctx context.Context, query string, limit int) []Chunk {
	var results []Chunk

	scan := func(rows *sql.Rows, srcType string) {
		if rows == nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var c Chunk
			c.SourceType = srcType
			rows.Scan(&c.ID, &c.Title, &c.Content, &c.URL, &c.Score)
			results = append(results, c)
		}
	}

	// FTS com keywords extraídas (mais preciso que a frase completa)
	kws := queryKeywords(query)
	if len(kws) > 0 {
		ftsQuery := strings.Join(kws, " OR ")

		// Publications com autores UFV no início do content
		pubRows, err := r.db.QueryContext(ctx, `
			SELECT p.id, p.title,
			       COALESCE('Pesquisadores UFV: ' || (
			           SELECT string_agg(r.full_name, ', ')
			           FROM publication_authors pa
			           JOIN researchers r ON r.id = pa.researcher_id
			           WHERE pa.publication_id = p.id
			         ) || '. ', '') ||
			         COALESCE(p.abstract,'') AS content,
			       NULL::text AS url, 0.5 AS score
			FROM publications p
			WHERE to_tsvector('portuguese', COALESCE(p.title,'') || ' ' || COALESCE(p.abstract,''))
			      @@ websearch_to_tsquery('portuguese', $1)
			ORDER BY p.cited_by_count DESC
			LIMIT $2`, ftsQuery, limit)
		if err == nil {
			scan(pubRows, "publication")
		}

		// Patents e opportunities (sem autores específicos)
		for _, tbl := range []struct{ table, title, content, url, srcType string }{
			{"patents", "COALESCE(title,'')", "abstract", "NULL::text", "patent"},
			{"opportunities", "title", "description", "url", "opportunity"},
		} {
			q := fmt.Sprintf(`
				SELECT id, %s, COALESCE(%s,'') as content, %s, 0.5 as score
				FROM %s
				WHERE to_tsvector('portuguese', COALESCE(%s,'') || ' ' || COALESCE(%s,''))
				      @@ websearch_to_tsquery('portuguese', $1)
				LIMIT $2`, tbl.title, tbl.content, tbl.url, tbl.table, tbl.title, tbl.content)
			rows, err := r.db.QueryContext(ctx, q, ftsQuery, limit)
			if err == nil {
				scan(rows, tbl.srcType)
			}
		}
	}

	// Fallback ILIKE por palavra-chave se FTS não trouxer nada
	if len(results) == 0 {
		for _, kw := range kws {
			like := "%" + kw + "%"
			rows, err := r.db.QueryContext(ctx, `
				SELECT id, title, COALESCE(abstract,'') as content,
				       NULL::text as url, 0.4 as score
				FROM publications
				WHERE title ILIKE $1 OR abstract ILIKE $1
				ORDER BY cited_by_count DESC
				LIMIT $2`, like, limit)
			if err == nil {
				scan(rows, "publication")
			}
			if len(results) >= limit {
				break
			}
		}
	}

	return results
}

// queryKeywords extrai termos relevantes (≥5 chars, sem stop words) para ILIKE fallback.
func queryKeywords(q string) []string {
	stop := map[string]bool{
		"quem": true, "qual": true, "quais": true, "como": true,
		"liste": true, "listar": true, "existe": true, "existem": true,
		"sobre": true, "entre": true, "para": true, "pela": true,
		"pelo": true, "mais": true, "menos": true, "quando": true,
		"onde": true, "esse": true, "esta": true, "este": true,
		"principais": true, "trabalhos": true, "pesquisa": true,
		"fazer": true, "feito": true, "sendo": true, "tendo": true,
	}
	seen := map[string]bool{}
	var words []string
	for _, w := range strings.Fields(strings.ToLower(q)) {
		w = strings.Trim(w, ".,?!;:()")
		if len(w) >= 5 && !stop[w] && !seen[w] {
			words = append(words, w)
			seen[w] = true
		}
	}
	return words
}

// rrf merges multiple result lists using Reciprocal Rank Fusion.
func rrf(k int, lists ...[]Chunk) []Chunk {
	const rrfK = 60
	type scored struct {
		chunk Chunk
		score float64
	}
	index := make(map[string]*scored)

	for _, list := range lists {
		for rank, chunk := range list {
			key := fmt.Sprintf("%s:%d", chunk.SourceType, chunk.ID)
			if _, ok := index[key]; !ok {
				c := chunk
				index[key] = &scored{chunk: c, score: 0}
			}
			index[key].score += 1.0 / float64(rrfK+rank+1)
		}
	}

	merged := make([]scored, 0, len(index))
	for _, s := range index {
		merged = append(merged, *s)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].score > merged[j].score
	})

	if k > len(merged) {
		k = len(merged)
	}
	result := make([]Chunk, k)
	for i := range result {
		result[i] = merged[i].chunk
	}
	return result
}

func urlEncode(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) {
			out = append(out, c)
		} else {
			out = append(out, '%')
			out = append(out, hexChar(c>>4))
			out = append(out, hexChar(c&0xf))
		}
	}
	return string(out)
}

func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~'
}

func hexChar(c byte) byte {
	if c < 10 {
		return '0' + c
	}
	return 'a' + c - 10
}

// Unused import guard
var _ = sql.ErrNoRows
