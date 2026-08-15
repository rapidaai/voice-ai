UPDATE public.assistant_conversation_metrics
SET
  name = 'duration',
  value = (value::numeric * 1000000)::bigint::text
WHERE name = 'conversation.duration_ms'
  AND value ~ '^[0-9]+(\.[0-9]+)?$';

UPDATE public.assistant_conversation_metrics
SET
  name = 'stt_duration',
  value = (value::numeric * 1000000)::bigint::text
WHERE name = 'stt.duration_ms'
  AND value ~ '^[0-9]+(\.[0-9]+)?$';

UPDATE public.assistant_conversation_metrics
SET
  name = 'tts_duration',
  value = (value::numeric * 1000000)::bigint::text
WHERE name = 'tts.duration_ms'
  AND value ~ '^[0-9]+(\.[0-9]+)?$';
