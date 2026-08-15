UPDATE assistant_conversation_message_metrics
SET name = 'eos_text_to_trigger_ms'
WHERE name = 'eos.trigger_ms';

UPDATE assistant_conversation_message_metrics
SET name = 'eos_word_count'
WHERE name = 'eos.word_count';

UPDATE assistant_conversation_message_metrics
SET name = 'eos_confidence'
WHERE name = 'eos.confidence';
