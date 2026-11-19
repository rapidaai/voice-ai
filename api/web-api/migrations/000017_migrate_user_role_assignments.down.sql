BEGIN;

LOCK TABLE public.role_definitions IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE public.user_organization_roles, public.user_project_roles IN ACCESS EXCLUSIVE MODE;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM public.user_organization_roles
        WHERE role_definition_id NOT IN (101, 102, 108)
    ) THEN
        RAISE EXCEPTION 'organization role assignment cannot be restored to a legacy role';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.user_project_roles
        WHERE role_definition_id NOT IN (109, 110, 112)
    ) THEN
        RAISE EXCEPTION 'project role assignment cannot be restored to a legacy role';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.user_organization_roles
        GROUP BY user_auth_id, organization_id, status
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'organization assignments exceed the legacy uniqueness contract';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.user_project_roles
        GROUP BY user_auth_id, project_id, status
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'project assignments exceed the legacy uniqueness contract';
    END IF;
END $$;

DROP INDEX public.user_organization_roles_current_assignment_idx;
DROP INDEX public.user_project_roles_current_assignment_idx;

ALTER TABLE public.user_organization_roles
    DROP CONSTRAINT user_organization_roles_role_definition_subject_fk,
    DROP CONSTRAINT user_organization_roles_role_definition_scope_fk,
    DROP CONSTRAINT user_organization_roles_role_subject_check,
    DROP CONSTRAINT user_organization_roles_role_scope_check,
    DROP COLUMN role_subject_type,
    DROP COLUMN role_scope_type;

ALTER TABLE public.user_project_roles
    DROP CONSTRAINT user_project_roles_role_definition_subject_fk,
    DROP CONSTRAINT user_project_roles_role_definition_scope_fk,
    DROP CONSTRAINT user_project_roles_role_subject_check,
    DROP CONSTRAINT user_project_roles_role_scope_check,
    DROP COLUMN role_subject_type,
    DROP COLUMN role_scope_type;

ALTER TABLE public.user_organization_roles
    RENAME COLUMN role_definition_id TO role;

ALTER TABLE public.user_organization_roles
    ALTER COLUMN role TYPE character varying(200) USING (
        CASE role
            WHEN 101 THEN 'owner'
            WHEN 102 THEN 'admin'
            WHEN 108 THEN 'member'
        END
    );

ALTER TABLE public.user_project_roles
    RENAME COLUMN role_definition_id TO role;

ALTER TABLE public.user_project_roles
    ALTER COLUMN role TYPE character varying(200) USING (
        CASE role
            WHEN 109 THEN 'admin'
            WHEN 110 THEN 'writer'
            WHEN 112 THEN 'reader'
        END
    );

ALTER TABLE public.user_organization_roles
    ADD CONSTRAINT user_organization_roles_user_org_status_key
        UNIQUE (user_auth_id, organization_id, status);

ALTER TABLE public.user_project_roles
    ADD CONSTRAINT user_project_roles_user_project_status_key
        UNIQUE (user_auth_id, project_id, status);

COMMIT;
