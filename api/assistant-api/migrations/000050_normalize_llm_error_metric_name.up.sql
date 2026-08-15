UPDATE assistant_conversation_message_metrics
SET name = 'agent.error',
    value = '1'
WHERE name = 'llm_error';
