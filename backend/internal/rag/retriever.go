package rag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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
	emb, err := r.embed(ctx, query)
	if err != nil {
		// Embed server not available — fall back to text-only search
		return r.textSearch(ctx, query, k), nil
	}

	vecResults := r.vectorSearch(ctx, emb, k*2)
	textResults := r.textSearch(ctx, query, k*2)

	return rrf(k, vecResults, textResults), nil
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

	// Publications
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, COALESCE(abstract,'') as content, NULL::text as url,
		       1 - (embedding <=> $1) AS score
		FROM publications
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1
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

func (r *Retriever) textSearch(ctx context.Context, query string, limit int) []Chunk {
	like := "%" + query + "%"
	var results []Chunk

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, COALESCE(abstract,'') as content,
		       NULL::text as url, 0.5 as score
		FROM publications
		WHERE title ILIKE $1 OR abstract ILIKE $1
		ORDER BY cited_by_count DESC
		LIMIT $2`, like, limit)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c Chunk
			c.SourceType = "publication"
			rows.Scan(&c.ID, &c.Title, &c.Content, &c.URL, &c.Score)
			results = append(results, c)
		}
	}

	rows2, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(title,'') as title,
		       COALESCE(abstract,'') as content, NULL::text as url, 0.5 as score
		FROM patents
		WHERE title ILIKE $1 OR abstract ILIKE $1
		LIMIT $2`, like, limit)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var c Chunk
			c.SourceType = "patent"
			rows2.Scan(&c.ID, &c.Title, &c.Content, &c.URL, &c.Score)
			results = append(results, c)
		}
	}

	rows3, err := r.db.QueryContext(ctx, `
		SELECT id, title, COALESCE(description,'') as content,
		       url, 0.5 as score
		FROM opportunities
		WHERE title ILIKE $1 OR description ILIKE $1
		LIMIT $2`, like, limit)
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
