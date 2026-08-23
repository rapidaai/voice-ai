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
