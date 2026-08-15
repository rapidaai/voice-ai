UPDATE assistant_conversation_message_metrics
SET name = 'tts.latency_ms'
WHERE name = 'tts_latency_ms';

UPDATE assistant_conversation_message_metrics
SET name = 'agent.latency_ms'
WHERE name = 'llm_latency_ms';

UPDATE assistant_conversation_message_metrics
SET name = 'eos.latency_ms'
WHERE name = 'eos_latency_ms';
