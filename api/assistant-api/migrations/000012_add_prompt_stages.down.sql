-- Drop assistant_prompt_stages table
DROP TABLE IF EXISTS public.assistant_prompt_stages;

-- Remove current_stage_id from assistant_conversations
ALTER TABLE public.assistant_conversations DROP COLUMN IF EXISTS current_stage_id;