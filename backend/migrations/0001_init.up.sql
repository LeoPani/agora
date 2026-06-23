-- Habilita pgvector para embeddings
CREATE EXTENSION IF NOT EXISTS vector;

-- Camada 1: Pesquisa
CREATE TABLE researchers (
    id BIGSERIAL PRIMARY KEY,
    openalex_id TEXT UNIQUE,
    orcid TEXT,
    full_name TEXT NOT NULL,
    normalized_name TEXT NOT NULL UNIQUE,
    department TEXT,
    institution TEXT DEFAULT 'UFV',
    embedding vector(384),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_researchers_dept ON researchers(department);
CREATE INDEX idx_researchers_embedding ON researchers
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE TABLE publications (
    id BIGSERIAL PRIMARY KEY,
    openalex_id TEXT UNIQUE,
    doi TEXT,
    title TEXT NOT NULL,
    abstract TEXT,
    publication_year INT,
    publication_date DATE,
    type TEXT,
    source TEXT,
    cited_by_count INT DEFAULT 0,
    topics JSONB,
    embedding vector(384),
    raw_data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_publications_year ON publications(publication_year);
CREATE INDEX idx_publications_embedding ON publications
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE TABLE publication_authors (
    publication_id BIGINT REFERENCES publications(id) ON DELETE CASCADE,
    researcher_id BIGINT REFERENCES researchers(id) ON DELETE CASCADE,
    author_position INT,
    is_corresponding BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (publication_id, researcher_id)
);

-- Camada 2: PI
CREATE TABLE patents (
    id BIGSERIAL PRIMARY KEY,
    inpi_number TEXT UNIQUE,
    lens_id TEXT,
    title TEXT NOT NULL,
    abstract TEXT,
    claims TEXT,
    description TEXT,
    ipc_codes TEXT[],
    ipc_section CHAR(1),
    filing_date DATE,
    publication_date DATE,
    grant_date DATE,
    legal_status TEXT,
    applicant TEXT NOT NULL,
    applicant_type TEXT,
    department TEXT,
    embedding vector(384),
    raw_data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_patents_ipc ON patents(ipc_section);
CREATE INDEX idx_patents_applicant_type ON patents(applicant_type);
CREATE INDEX idx_patents_embedding ON patents
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE TABLE patent_inventors (
    patent_id BIGINT REFERENCES patents(id) ON DELETE CASCADE,
    researcher_id BIGINT REFERENCES researchers(id) ON DELETE CASCADE,
    PRIMARY KEY (patent_id, researcher_id)
);

CREATE TABLE patent_citations (
    patent_id BIGINT REFERENCES patents(id) ON DELETE CASCADE,
    publication_id BIGINT REFERENCES publications(id) ON DELETE CASCADE,
    source TEXT,
    PRIMARY KEY (patent_id, publication_id)
);

-- Camada 3: Mercado
CREATE TABLE opportunities (
    id BIGSERIAL PRIMARY KEY,
    source TEXT NOT NULL,
    external_id TEXT,
    title TEXT NOT NULL,
    description TEXT,
    url TEXT,
    opening_date DATE,
    closing_date DATE,
    max_value_brl NUMERIC(15,2),
    areas TEXT[],
    opportunity_type TEXT,
    status TEXT DEFAULT 'open',
    embedding vector(384),
    raw_data JSONB,
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (source, external_id)
);
CREATE INDEX idx_opportunities_status ON opportunities(status);
CREATE INDEX idx_opportunities_closing ON opportunities(closing_date);
CREATE INDEX idx_opportunities_embedding ON opportunities
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Camada 4: Parceiros
CREATE TABLE partners (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL UNIQUE,
    cnpj TEXT,
    partner_type TEXT,
    sector TEXT,
    location TEXT,
    n_patents INT DEFAULT 0,
    n_citations_to_ufv INT DEFAULT 0,
    raw_data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Sinais gerados pelo radar
CREATE TABLE signals (
    id BIGSERIAL PRIMARY KEY,
    signal_type TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    score NUMERIC(4,3),
    relevance TEXT,
    entities JSONB,
    reasoning JSONB,
    status TEXT DEFAULT 'new',
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);
CREATE INDEX idx_signals_type ON signals(signal_type);
CREATE INDEX idx_signals_status ON signals(status);
CREATE INDEX idx_signals_score ON signals(score DESC);

-- Logs de coleta
CREATE TABLE collector_runs (
    id BIGSERIAL PRIMARY KEY,
    collector_name TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    status TEXT,
    records_collected INT DEFAULT 0,
    error_message TEXT
);
