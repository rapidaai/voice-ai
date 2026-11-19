BEGIN;

LOCK TABLE public.role_definitions IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE public.user_organization_roles, public.user_project_roles IN ACCESS EXCLUSIVE MODE;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM public.user_organization_roles
        WHERE role NOT IN ('owner', 'admin', 'member')
    ) THEN
        RAISE EXCEPTION 'unknown organization role value';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.user_project_roles
        WHERE role NOT IN ('super admin', 'admin', 'writer', 'reader')
    ) THEN
        RAISE EXCEPTION 'unknown project role value';
    END IF;

    IF (
        SELECT count(*)
        FROM public.role_definitions
        WHERE (id, key, scope_type, subject_type, definition_type, status) IN (
            (101, 'organization_owner', 'ORGANIZATION', 'USER', 'BUILT_IN', 'ACTIVE'),
            (102, 'organization_admin', 'ORGANIZATION', 'USER', 'BUILT_IN', 'ACTIVE'),
            (108, 'organization_member', 'ORGANIZATION', 'USER', 'BUILT_IN', 'ACTIVE'),
            (109, 'project_admin', 'PROJECT', 'USER', 'BUILT_IN', 'ACTIVE'),
            (110, 'project_developer', 'PROJECT', 'USER', 'BUILT_IN', 'ACTIVE'),
            (112, 'project_viewer', 'PROJECT', 'USER', 'BUILT_IN', 'ACTIVE')
        ) AND organization_id IS NULL
    ) <> 6 THEN
        RAISE EXCEPTION 'user role catalog prerequisites are unavailable';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.user_organization_roles
        WHERE status IN ('ACTIVE', 'INVITED')
        GROUP BY user_auth_id, organization_id,
            CASE role WHEN 'owner' THEN 101 WHEN 'admin' THEN 102 WHEN 'member' THEN 108 END
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate current organization role assignment';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.user_project_roles
        WHERE status IN ('ACTIVE', 'INVITED')
        GROUP BY user_auth_id, project_id,
            CASE role
                WHEN 'super admin' THEN 109
                WHEN 'admin' THEN 109
                WHEN 'writer' THEN 110
                WHEN 'reader' THEN 112
            END
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate current project role assignment';
    END IF;
END $$;

UPDATE public.user_organization_roles
SET role = CASE role
    WHEN 'owner' THEN '101'
    WHEN 'admin' THEN '102'
    WHEN 'member' THEN '108'
END;

UPDATE public.user_project_roles
SET role = CASE role
    WHEN 'super admin' THEN '109'
    WHEN 'admin' THEN '109'
    WHEN 'writer' THEN '110'
    WHEN 'reader' THEN '112'
END;

ALTER TABLE public.user_organization_roles
    DROP CONSTRAINT user_organization_roles_user_org_status_key,
    ALTER COLUMN role TYPE bigint USING role::bigint;

ALTER TABLE public.user_organization_roles
    RENAME COLUMN role TO role_definition_id;

ALTER TABLE public.user_project_roles
    DROP CONSTRAINT user_project_roles_user_project_status_key,
    ALTER COLUMN role TYPE bigint USING role::bigint;

ALTER TABLE public.user_project_roles
    RENAME COLUMN role TO role_definition_id;

ALTER TABLE public.user_organization_roles
    ADD COLUMN role_scope_type character varying(32),
    ADD COLUMN role_subject_type character varying(32);

UPDATE public.user_organization_roles
SET role_scope_type = 'ORGANIZATION',
    role_subject_type = 'USER';

ALTER TABLE public.user_project_roles
    ADD COLUMN role_scope_type character varying(32),
    ADD COLUMN role_subject_type character varying(32);

UPDATE public.user_project_roles
SET role_scope_type = 'PROJECT',
    role_subject_type = 'USER';

ALTER TABLE public.user_organization_roles
    ALTER COLUMN role_scope_type SET NOT NULL,
    ALTER COLUMN role_subject_type SET NOT NULL,
    ADD CONSTRAINT user_organization_roles_role_scope_check
        CHECK (role_scope_type = 'ORGANIZATION'),
    ADD CONSTRAINT user_organization_roles_role_subject_check
        CHECK (role_subject_type = 'USER'),
    ADD CONSTRAINT user_organization_roles_role_definition_scope_fk
        FOREIGN KEY (role_definition_id, role_scope_type)
        REFERENCES public.role_definitions (id, scope_type) ON DELETE RESTRICT,
    ADD CONSTRAINT user_organization_roles_role_definition_subject_fk
        FOREIGN KEY (role_definition_id, role_subject_type)
        REFERENCES public.role_definitions (id, subject_type) ON DELETE RESTRICT;

ALTER TABLE public.user_project_roles
    ALTER COLUMN role_scope_type SET NOT NULL,
    ALTER COLUMN role_subject_type SET NOT NULL,
    ADD CONSTRAINT user_project_roles_role_scope_check
        CHECK (role_scope_type = 'PROJECT'),
    ADD CONSTRAINT user_project_roles_role_subject_check
        CHECK (role_subject_type = 'USER'),
    ADD CONSTRAINT user_project_roles_role_definition_scope_fk
        FOREIGN KEY (role_definition_id, role_scope_type)
        REFERENCES public.role_definitions (id, scope_type) ON DELETE RESTRICT,
    ADD CONSTRAINT user_project_roles_role_definition_subject_fk
        FOREIGN KEY (role_definition_id, role_subject_type)
        REFERENCES public.role_definitions (id, subject_type) ON DELETE RESTRICT;

CREATE UNIQUE INDEX user_organization_roles_current_assignment_idx
    ON public.user_organization_roles (user_auth_id, organization_id, role_definition_id)
    WHERE status IN ('ACTIVE', 'INVITED');

CREATE UNIQUE INDEX user_project_roles_current_assignment_idx
    ON public.user_project_roles (user_auth_id, project_id, role_definition_id)
    WHERE status IN ('ACTIVE', 'INVITED');

COMMIT;
