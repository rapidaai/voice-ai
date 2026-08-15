UPDATE assistant_conversation_message_metrics
SET name = 'tts_latency_ms'
WHERE name = 'tts.latency_ms';

UPDATE assistant_conversation_message_metrics
SET name = 'llm_latency_ms'
WHERE name = 'agent.latency_ms';

UPDATE assistant_conversation_message_metrics
SET name = 'eos_latency_ms'
WHERE name = 'eos.latency_ms';
