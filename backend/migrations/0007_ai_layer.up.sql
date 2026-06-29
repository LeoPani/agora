-- AI Layer: observabilidade de LLM, conversas RAG, sinais e rascunhos do agente

-- Logs de todas as chamadas LLM
CREATE TABLE IF NOT EXISTS llm_calls (
    id               BIGSERIAL PRIMARY KEY,
    purpose          TEXT NOT NULL,
    provider         TEXT NOT NULL,
    model            TEXT NOT NULL,
    prompt_tokens    INT,
    completion_tokens INT,
    total_tokens     INT,
    cost_usd         NUMERIC(10,6),
    latency_ms       INT,
    success          BOOLEAN NOT NULL DEFAULT TRUE,
    error_message    TEXT,
    metadata         JSONB,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_llm_calls_purpose ON llm_calls(purpose);
CREATE INDEX IF NOT EXISTS idx_llm_calls_created  ON llm_calls(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_llm_calls_provider ON llm_calls(provider);

-- Histórico de conversas RAG (Oráculo)
CREATE TABLE IF NOT EXISTS conversations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT,
    title      TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversation_messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('user','assistant','system')),
    content         TEXT NOT NULL,
    sources         JSONB,
    llm_call_id     BIGINT REFERENCES llm_calls(id),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_conv ON conversation_messages(conversation_id, created_at);

-- Sinais gerados pelo radar (pré-requisito para rascunhos do agente)
CREATE TABLE IF NOT EXISTS signals (
    id          BIGSERIAL PRIMARY KEY,
    signal_type TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT,
    score       NUMERIC(5,4) DEFAULT 0,
    entities    JSONB,
    metadata    JSONB,
    status      TEXT NOT NULL DEFAULT 'new',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_signals_type   ON signals(signal_type);
CREATE INDEX IF NOT EXISTS idx_signals_score  ON signals(score DESC);
CREATE INDEX IF NOT EXISTS idx_signals_status ON signals(status);

-- Rascunhos gerados pelo agente de transferência de tecnologia
CREATE TABLE IF NOT EXISTS agent_drafts (
    id           BIGSERIAL PRIMARY KEY,
    signal_id    BIGINT REFERENCES signals(id) ON DELETE CASCADE,
    draft_type   TEXT NOT NULL DEFAULT 'email_intro',
    subject      TEXT,
    body         TEXT NOT NULL,
    context_used JSONB,
    status       TEXT NOT NULL DEFAULT 'draft',
    cost_usd     NUMERIC(10,6),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    reviewed_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_drafts_signal ON agent_drafts(signal_id);
CREATE INDEX IF NOT EXISTS idx_drafts_status ON agent_drafts(status);

-- Coluna de embedding para oportunidades (busca semântica no Oráculo)
ALTER TABLE opportunities ADD COLUMN IF NOT EXISTS embedding vector(384);
