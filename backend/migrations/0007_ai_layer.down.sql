ALTER TABLE opportunities DROP COLUMN IF EXISTS embedding;
DROP TABLE IF EXISTS agent_drafts;
DROP TABLE IF EXISTS signals;
DROP TABLE IF EXISTS conversation_messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS llm_calls;
