UPDATE assistant_conversation_message_metrics
SET name = 'llm_error'
WHERE name = 'agent.error';
