SET lock_timeout = '5s';

ALTER TABLE public.notification_settings
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.organizations
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.project_credentials
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.projects
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.user_auth_tokens
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.user_auths
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.user_feature_permissions
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.user_organization_roles
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.user_project_roles
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.user_roles
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.vaults
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

RESET lock_timeout;
