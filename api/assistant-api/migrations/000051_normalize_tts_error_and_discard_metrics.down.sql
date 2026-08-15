UPDATE assistant_conversation_message_metrics
SET name = 'tts_error'
WHERE name = 'tts.error';

UPDATE assistant_conversation_message_metrics
SET name = 'discarded_tts_chunk',
    value = 'true'
WHERE name = 'tts.discard_chunk_count';
