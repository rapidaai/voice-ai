DO \\$ BEGIN RAISE EXCEPTION 'actor constraints and immutability triggers are irreversible; restore the complete pre-migration backup set instead'; END \\$;
