UPDATE public.assistant_conversation_metrics
SET
  name = 'conversation.duration_ms',
  value = FLOOR(value::numeric / 1000000)::bigint::text
WHERE name = 'duration'
  AND value ~ '^[0-9]+(\.[0-9]+)?$';

UPDATE public.assistant_conversation_metrics
SET
  name = 'stt.duration_ms',
  value = FLOOR(value::numeric / 1000000)::bigint::text
WHERE name = 'stt_duration'
  AND value ~ '^[0-9]+(\.[0-9]+)?$';

UPDATE public.assistant_conversation_metrics
SET
  name = 'tts.duration_ms',
  value = FLOOR(value::numeric / 1000000)::bigint::text
WHERE name = 'tts_duration'
  AND value ~ '^[0-9]+(\.[0-9]+)?$';
