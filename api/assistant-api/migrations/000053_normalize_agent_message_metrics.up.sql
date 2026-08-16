UPDATE assistant_conversation_message_metrics
SET name = 'agent.message_char_count'
WHERE name = 'llm_input_char_count';

UPDATE assistant_conversation_message_metrics
SET name = 'agent.message_count'
WHERE name = 'llm_history_count';

UPDATE assistant_conversation_message_metrics
SET name = 'agent.response_char_count'
WHERE name = 'llm_response_char_count';
