DROP TABLE IF EXISTS interested_matches;
ALTER TABLE partners
    DROP COLUMN IF EXISTS linkedin_url,
    DROP COLUMN IF EXISTS lattes_id,
    DROP COLUMN IF EXISTS cnae_code,
    DROP COLUMN IF EXISTS interest_score,
    DROP COLUMN IF EXISTS contact_email,
    DROP COLUMN IF EXISTS source;
