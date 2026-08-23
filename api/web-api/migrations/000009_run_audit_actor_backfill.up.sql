UPDATE public.notification_settings
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.organizations
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.project_credentials
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.projects
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.user_auth_tokens
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.user_auths
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.user_feature_permissions
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.user_organization_roles
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.user_project_roles
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.user_roles
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.vaults
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;
