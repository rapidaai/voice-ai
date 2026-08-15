UPDATE assistant_conversation_message_metrics
SET name = 'agent_total_token'
WHERE name = 'agent.total_token';

UPDATE assistant_conversation_message_metrics
SET name = 'agent_cached_content_token'
WHERE name = 'agent.cached_content_token';

UPDATE assistant_conversation_message_metrics
SET name = 'agent_cost'
WHERE name = 'agent.cost';

UPDATE assistant_conversation_message_metrics
SET name = 'agent_input_cost'
WHERE name = 'agent.input_cost';

UPDATE assistant_conversation_message_metrics
SET name = 'agent_output_cost'
WHERE name = 'agent.output_cost';

UPDATE assistant_conversation_message_metrics
SET name = 'agent_llm_request_id'
WHERE name = 'agent.llm_request_id';

UPDATE assistant_conversation_message_metrics
SET name = 'agent_token_pre_second'
WHERE name = 'agent.token_pre_second';
