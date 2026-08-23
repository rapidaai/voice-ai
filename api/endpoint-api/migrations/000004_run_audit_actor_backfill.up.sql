SET lock_timeout = '5s';

DO $migration$
BEGIN
    IF EXISTS (SELECT 1 FROM public.endpoint_cachings WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.endpoint_log_arguments WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.endpoint_log_metadata WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.endpoint_log_metrics WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.endpoint_log_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.endpoint_provider_model_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.endpoint_provider_models WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.endpoint_retries WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.endpoint_tags WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.endpoints WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
    THEN
        RAISE EXCEPTION 'endpoint-api audit actor backfill requires positive user IDs';
    END IF;
END
$migration$;

UPDATE public.endpoint_cachings
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.endpoint_log_arguments
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.endpoint_log_metadata
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.endpoint_log_metrics
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.endpoint_log_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.endpoint_provider_model_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.endpoint_provider_models
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.endpoint_retries
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.endpoint_tags
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.endpoints
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

RESET lock_timeout;
