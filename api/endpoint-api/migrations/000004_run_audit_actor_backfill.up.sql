UPDATE public.endpoint_cachings
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.endpoint_log_arguments
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.endpoint_log_metadata
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.endpoint_log_metrics
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.endpoint_log_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.endpoint_provider_model_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.endpoint_provider_models
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.endpoint_retries
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.endpoint_tags
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.endpoints
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;
