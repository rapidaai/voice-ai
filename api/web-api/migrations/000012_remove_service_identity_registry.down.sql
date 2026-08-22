DO $migration$
BEGIN
    RAISE EXCEPTION 'service identity registry cleanup is irreversible; restore the complete pre-cleanup backup set instead';
END
$migration$;
