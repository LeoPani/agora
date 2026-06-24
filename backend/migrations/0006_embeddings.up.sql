-- Embeddings semânticos para busca por similaridade via pgvector
-- Modelo: paraphrase-multilingual-MiniLM-L12-v2 (384 dims)

ALTER TABLE publications ADD COLUMN IF NOT EXISTS embedding vector(384);
ALTER TABLE patents      ADD COLUMN IF NOT EXISTS embedding vector(384);

-- HNSW index — busca aproximada rápida (cosine)
CREATE INDEX IF NOT EXISTS idx_pub_embedding
    ON publications USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX IF NOT EXISTS idx_pat_embedding
    ON patents USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Tabela de controle de embedding
CREATE TABLE IF NOT EXISTS embedding_runs (
    id           BIGSERIAL PRIMARY KEY,
    entity_type  TEXT NOT NULL,  -- 'publication' | 'patent'
    model        TEXT NOT NULL DEFAULT 'paraphrase-multilingual-MiniLM-L12-v2',
    total        INT  NOT NULL DEFAULT 0,
    embedded     INT  NOT NULL DEFAULT 0,
    started_at   TIMESTAMPTZ DEFAULT NOW(),
    finished_at  TIMESTAMPTZ
);
