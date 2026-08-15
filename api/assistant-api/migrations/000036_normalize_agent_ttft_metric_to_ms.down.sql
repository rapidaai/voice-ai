UPDATE public.assistant_conversation_message_metrics
SET
  name = 'agent_time_to_first_token',
  value = (value::numeric * 1000000)::bigint::text
WHERE name = 'agent.ttft_ms'
  AND value ~ '^[0-9]+(\.[0-9]+)?$';
