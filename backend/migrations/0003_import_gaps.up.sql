CREATE TABLE import_gaps (
    id BIGSERIAL PRIMARY KEY,
    sh4_code TEXT NOT NULL,
    description TEXT,
    country_origin TEXT,
    year INT NOT NULL,
    import_value_usd NUMERIC(18,2),
    import_kg NUMERIC(18,2),
    ufv_related_areas TEXT[],
    opportunity_score NUMERIC(5,4),
    raw_data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(sh4_code, country_origin, year)
);
CREATE INDEX idx_import_gaps_sh4 ON import_gaps(sh4_code);
CREATE INDEX idx_import_gaps_score ON import_gaps(opportunity_score DESC);
CREATE INDEX idx_import_gaps_year ON import_gaps(year);
