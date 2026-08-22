SET lock_timeout = '5s';

ALTER TABLE public.external_audit_metadata
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.external_audits
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

RESET lock_timeout;
