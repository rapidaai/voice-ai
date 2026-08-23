SET lock_timeout = '5s';

ALTER TABLE public.notification_settings
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.organizations
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.project_credentials
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.projects
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.user_auth_tokens
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.user_auths
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.user_feature_permissions
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.user_organization_roles
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.user_project_roles
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.user_roles
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

ALTER TABLE public.vaults
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id bigint,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id bigint;

RESET lock_timeout;
