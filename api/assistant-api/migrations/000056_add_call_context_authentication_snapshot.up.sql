SET lock_timeout = '5s';

ALTER TABLE public.call_contexts
    ADD COLUMN auth_user_id bigint,
    ADD COLUMN auth_actor_type varchar(50),
    ADD COLUMN auth_actor_id varchar(255);

RESET lock_timeout;
