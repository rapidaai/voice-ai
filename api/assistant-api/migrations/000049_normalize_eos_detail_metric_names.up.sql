UPDATE assistant_conversation_message_metrics
SET name = 'eos.trigger_ms'
WHERE name = 'eos_text_to_trigger_ms';

UPDATE assistant_conversation_message_metrics
SET name = 'eos.word_count'
WHERE name = 'eos_word_count';

UPDATE assistant_conversation_message_metrics
SET name = 'eos.confidence'
WHERE name = 'eos_confidence';

DELETE FROM assistant_conversation_message_metrics
WHERE name = 'eos_char_count';
