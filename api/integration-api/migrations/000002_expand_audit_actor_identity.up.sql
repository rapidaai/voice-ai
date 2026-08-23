SET lock_timeout = '5s';

ALTER TABLE public.external_audits
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.external_audit_metadata
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

RESET lock_timeout;
