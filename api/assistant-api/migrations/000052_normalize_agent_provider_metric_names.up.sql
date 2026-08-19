UPDATE assistant_conversation_message_metrics
SET name = 'agent.total_token'
WHERE name IN ('agent_total_token', 'total_token');

UPDATE assistant_conversation_message_metrics
SET name = 'agent.cached_content_token'
WHERE name IN ('agent_cached_content_token', 'cached_content_token');

UPDATE assistant_conversation_message_metrics
SET name = 'agent.cost'
WHERE name IN ('agent_cost', 'cost');

UPDATE assistant_conversation_message_metrics
SET name = 'agent.input_cost'
WHERE name IN ('agent_input_cost', 'input_cost');

UPDATE assistant_conversation_message_metrics
SET name = 'agent.output_cost'
WHERE name IN ('agent_output_cost', 'output_cost');

UPDATE assistant_conversation_message_metrics
SET name = 'agent.llm_request_id'
WHERE name IN ('agent_llm_request_id', 'llm_request_id');

UPDATE assistant_conversation_message_metrics
SET name = 'agent.token_pre_second'
WHERE name IN ('agent_token_pre_second', 'token_pre_second');

DELETE FROM assistant_conversation_message_metrics
WHERE name IN (
  'time_taken',
  'agent_time_taken',
  'time_to_first_token',
  'provider_total_time',
  'agent_provider_total_time',
  'provider_generate_time',
  'agent_provider_generate_time'
);
