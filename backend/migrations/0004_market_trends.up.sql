CREATE TABLE market_trends (
    id BIGSERIAL PRIMARY KEY,
    keyword TEXT NOT NULL,
    geo TEXT NOT NULL DEFAULT 'BR',
    timeframe TEXT NOT NULL,
    avg_interest INT,
    peak_interest INT,
    growth_pct NUMERIC(8,2),
    related_queries JSONB,
    related_topics JSONB,
    ufv_department TEXT,
    raw_data JSONB,
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(keyword, geo, timeframe)
);
CREATE INDEX idx_market_trends_keyword ON market_trends(keyword);
CREATE INDEX idx_market_trends_growth ON market_trends(growth_pct DESC);
CREATE INDEX idx_market_trends_dept ON market_trends(ufv_department);
