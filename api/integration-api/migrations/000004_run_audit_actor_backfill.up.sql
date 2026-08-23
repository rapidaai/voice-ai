SET lock_timeout = '5s';

UPDATE public.external_audits
SET created_actor_type = 'unknown',
    created_actor_id = NULL,
    updated_actor_type = NULL,
    updated_actor_id = NULL;

UPDATE public.external_audit_metadata
SET created_actor_type = 'unknown',
    created_actor_id = NULL,
    updated_actor_type = NULL,
    updated_actor_id = NULL;

RESET lock_timeout;
