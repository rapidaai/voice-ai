UPDATE public.assistant_conversation_metrics
SET name = 'agent.init_ms'
WHERE name = 'llm_init_ms';
