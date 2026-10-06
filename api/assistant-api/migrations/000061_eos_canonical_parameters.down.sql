DO $$
BEGIN
    RAISE EXCEPTION 'EOS parameter migration is forward-only. Restore the pre-migration backup and matching migration version metadata before deploying older code.';
END $$;
