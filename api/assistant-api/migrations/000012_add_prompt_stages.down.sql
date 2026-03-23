-- Drop assistant_prompt_stages table
DROP TABLE IF EXISTS assistant_prompt_stages;

-- Remove current_stage_id from assistant_conversations
ALTER TABLE assistant_conversations DROP COLUMN IF EXISTS current_stage_id;