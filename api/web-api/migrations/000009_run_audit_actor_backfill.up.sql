SET lock_timeout = '5s';

DO $migration$
BEGIN
    IF EXISTS (SELECT 1 FROM public.notification_settings WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.organizations WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.project_credentials WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.projects WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.user_auth_tokens WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.user_auths WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.user_feature_permissions WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.user_organization_roles WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.user_project_roles WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.user_roles WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.vaults WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
    THEN
        RAISE EXCEPTION 'web-api audit actor backfill requires positive user IDs';
    END IF;
END
$migration$;

UPDATE public.notification_settings
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.organizations
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.project_credentials
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.projects
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.user_auth_tokens
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.user_auths
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.user_feature_permissions
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.user_organization_roles
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.user_project_roles
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.user_roles
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.vaults
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

RESET lock_timeout;
