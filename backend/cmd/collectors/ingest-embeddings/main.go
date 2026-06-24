package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

const dbURL = "postgresql://agora:agora_dev@localhost:5433/agora?sslmode=disable"

type EmbRow struct {
	ID        int64     `json:"id"`
	Embedding []float32 `json:"embedding"`
}

func ingestFile(db *sql.DB, path string, table string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	stmt, err := tx.Prepare(fmt.Sprintf(
		`UPDATE %s SET embedding = $1 WHERE id = $2`, table))
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	defer stmt.Close()

	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for sc.Scan() {
		var row EmbRow
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			continue
		}
		vec := pgvector.NewVector(row.Embedding)
		if _, err := stmt.Exec(vec, row.ID); err != nil {
			tx.Rollback()
			return n, fmt.Errorf("update %s id=%d: %w", table, row.ID, err)
		}
		n++
		if n%1000 == 0 {
			slog.Info("ingest-embeddings progress", "table", table, "n", n)
		}
	}

	return n, tx.Commit()
}

func main() {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		slog.Error("db open", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	dataDir := "../ai-service/data"

	files := map[string]string{
		dataDir + "/embeddings_publications.jsonl": "publications",
		dataDir + "/embeddings_patents.jsonl":      "patents",
	}

	total := 0
	for path, table := range files {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			slog.Info("file not found, skipping", "path", path)
			continue
		}
		n, err := ingestFile(db, path, table)
		if err != nil {
			slog.Error("ingest failed", "path", path, "err", err)
			os.Exit(1)
		}
		slog.Info("ingest done", "table", table, "n", n)

		// Registra o run
		model := "paraphrase-multilingual-MiniLM-L12-v2"
		if strings.Contains(path, "patent") {
			model = "paraphrase-multilingual-MiniLM-L12-v2"
		}
		db.Exec(`INSERT INTO embedding_runs (entity_type, model, total, embedded, finished_at)
		         VALUES ($1, $2, $3, $3, NOW())
		         ON CONFLICT DO NOTHING`, table, model, n)

		total += n
	}

	slog.Info("ingest-embeddings complete", "total", total)
}
