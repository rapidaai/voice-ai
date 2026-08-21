SET lock_timeout = '5s';

ALTER TABLE public.endpoint_cachings
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.endpoint_log_arguments
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.endpoint_log_metadata
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.endpoint_log_metrics
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.endpoint_log_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.endpoint_provider_model_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.endpoint_provider_models
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.endpoint_retries
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.endpoint_tags
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.endpoints
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

RESET lock_timeout;
