DO $migration$
BEGIN
    RAISE EXCEPTION 'legacy audit cleanup is irreversible; restore the complete pre-cleanup backup set instead';
END
$migration$;
