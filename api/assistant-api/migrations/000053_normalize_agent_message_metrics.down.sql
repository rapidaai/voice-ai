UPDATE assistant_conversation_message_metrics
SET name = 'llm_input_char_count'
WHERE name = 'agent.message_char_count';

UPDATE assistant_conversation_message_metrics
SET name = 'llm_history_count'
WHERE name = 'agent.message_count';

UPDATE assistant_conversation_message_metrics
SET name = 'llm_response_char_count'
WHERE name = 'agent.response_char_count';
