SET lock_timeout = '5s';

ALTER TABLE public.endpoint_cachings
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

ALTER TABLE public.endpoint_log_arguments
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

ALTER TABLE public.endpoint_log_metadata
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

ALTER TABLE public.endpoint_log_metrics
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

ALTER TABLE public.endpoint_log_options
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

ALTER TABLE public.endpoint_provider_model_options
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

ALTER TABLE public.endpoint_provider_models
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

ALTER TABLE public.endpoint_retries
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

ALTER TABLE public.endpoint_tags
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

ALTER TABLE public.endpoints
    DROP COLUMN created_by,
    DROP COLUMN updated_by;

RESET lock_timeout;
