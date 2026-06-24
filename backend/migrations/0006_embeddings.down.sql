DROP TABLE IF EXISTS embedding_runs;
DROP INDEX IF EXISTS idx_pub_embedding;
DROP INDEX IF EXISTS idx_pat_embedding;
ALTER TABLE publications DROP COLUMN IF EXISTS embedding;
ALTER TABLE patents      DROP COLUMN IF EXISTS embedding;
