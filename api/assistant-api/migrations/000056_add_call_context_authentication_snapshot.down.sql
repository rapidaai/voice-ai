SET lock_timeout = '5s';

ALTER TABLE public.call_contexts
    DROP COLUMN auth_user_id,
    DROP COLUMN auth_actor_type,
    DROP COLUMN auth_actor_id;

RESET lock_timeout;
