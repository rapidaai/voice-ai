UPDATE assistant_conversation_message_metrics
SET name = 'tts.error',
    value = '1'
WHERE name = 'tts_error';

UPDATE assistant_conversation_message_metrics
SET name = 'tts.discard_chunk_count',
    value = '1'
WHERE name = 'discarded_tts_chunk';

DELETE FROM assistant_conversation_message_metrics
WHERE name = 'discarded_tts';
