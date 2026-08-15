UPDATE public.assistant_conversation_metrics
SET name = 'llm_init_ms'
WHERE name = 'agent.init_ms';
