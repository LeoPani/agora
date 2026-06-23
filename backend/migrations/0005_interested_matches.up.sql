-- Vincula parceiros potenciais a patentes/publicações específicas
CREATE TABLE interested_matches (
    id BIGSERIAL PRIMARY KEY,
    partner_id BIGINT REFERENCES partners(id) ON DELETE CASCADE,
    patent_id BIGINT REFERENCES patents(id) ON DELETE SET NULL,
    publication_id BIGINT REFERENCES publications(id) ON DELETE SET NULL,
    match_reason TEXT NOT NULL,  -- 'cites_patent','cites_pub','same_topic','same_cnae','lattes_match'
    confidence NUMERIC(4,3) DEFAULT 0.5,
    source TEXT NOT NULL,        -- 'cnpj','lattes','citation','comex'
    raw_data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(partner_id, patent_id, publication_id, match_reason)
);
CREATE INDEX idx_interested_matches_partner ON interested_matches(partner_id);
CREATE INDEX idx_interested_matches_patent  ON interested_matches(patent_id);
CREATE INDEX idx_interested_matches_source  ON interested_matches(source);
CREATE INDEX idx_interested_matches_conf    ON interested_matches(confidence DESC);

-- Atualiza partners com campos extra para interessados
ALTER TABLE partners
    ADD COLUMN IF NOT EXISTS linkedin_url TEXT,
    ADD COLUMN IF NOT EXISTS lattes_id TEXT,
    ADD COLUMN IF NOT EXISTS cnae_code TEXT,
    ADD COLUMN IF NOT EXISTS interest_score NUMERIC(5,4) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS contact_email TEXT,
    ADD COLUMN IF NOT EXISTS source TEXT DEFAULT 'manual';

CREATE INDEX IF NOT EXISTS idx_partners_interest ON partners(interest_score DESC);
CREATE INDEX IF NOT EXISTS idx_partners_source   ON partners(source);
