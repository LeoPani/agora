CREATE TABLE research_groups (
    id BIGSERIAL PRIMARY KEY,
    dgp_id TEXT UNIQUE,
    name TEXT NOT NULL,
    leader TEXT,
    department TEXT,
    research_lines TEXT[],
    main_area TEXT,
    formation_year INT,
    institution TEXT DEFAULT 'UFV',
    raw_data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_research_groups_dept ON research_groups(department);
CREATE INDEX idx_research_groups_area ON research_groups(main_area);

CREATE TABLE research_group_members (
    group_id     BIGINT REFERENCES research_groups(id) ON DELETE CASCADE,
    researcher_id BIGINT REFERENCES researchers(id) ON DELETE CASCADE,
    role TEXT,
    PRIMARY KEY (group_id, researcher_id)
);
