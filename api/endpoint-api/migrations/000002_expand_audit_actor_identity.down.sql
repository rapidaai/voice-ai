SET lock_timeout = '5s';

ALTER TABLE public.endpoint_cachings
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.endpoint_log_arguments
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.endpoint_log_metadata
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.endpoint_log_metrics
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.endpoint_log_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.endpoint_provider_model_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.endpoint_provider_models
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.endpoint_retries
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.endpoint_tags
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.endpoints
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

RESET lock_timeout;
