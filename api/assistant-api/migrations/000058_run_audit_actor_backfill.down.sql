DO \\$ BEGIN RAISE EXCEPTION 'actor conversion is irreversible; restore the complete pre-migration backup set instead'; END \\$;
