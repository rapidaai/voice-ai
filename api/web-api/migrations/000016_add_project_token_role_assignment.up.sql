BEGIN;

LOCK TABLE public.project_credentials IN ACCESS EXCLUSIVE MODE;

ALTER TABLE public.role_definitions
    ADD CONSTRAINT role_definitions_subject_reference_unique UNIQUE (id, subject_type);

INSERT INTO public.role_definitions (
    organization_id, key, name, description, scope_type, subject_type, definition_type,
    status, created_actor_type, created_actor_id
) VALUES (
    NULL, 'project_token_full_access', 'Project Token Full Access',
    'Grants a project token the full Project permission bundle captured when project-token role assignment was introduced.',
    'PROJECT', 'PROJECT_TOKEN', 'BUILT_IN', 'ACTIVE', 'unknown', NULL
);

INSERT INTO public.role_permissions (
    role_definition_id, permission_definition_id, scope_type,
    status, created_actor_type, created_actor_id
)
SELECT
    role_definition.id, permission_definition.id, 'PROJECT', 'ACTIVE', 'unknown', NULL
FROM public.role_definitions role_definition
CROSS JOIN public.permission_definitions permission_definition
WHERE role_definition.organization_id IS NULL
  AND role_definition.key = 'project_token_full_access'
  AND role_definition.definition_type = 'BUILT_IN'
  AND permission_definition.scope_type = 'PROJECT'
  AND permission_definition.status = 'ACTIVE'
ORDER BY permission_definition.id;

ALTER TABLE public.project_credentials
    ADD COLUMN role_definition_id bigint,
    ADD COLUMN role_subject_type character varying(32);

UPDATE public.project_credentials
SET role_definition_id = (
        SELECT id
        FROM public.role_definitions
        WHERE organization_id IS NULL
          AND key = 'project_token_full_access'
          AND definition_type = 'BUILT_IN'
    ),
    role_subject_type = 'PROJECT_TOKEN';

DO $$
BEGIN
    IF (SELECT count(*) FROM public.role_definitions WHERE organization_id IS NULL
        AND key = 'project_token_full_access'
        AND name = 'Project Token Full Access'
        AND description = 'Grants a project token the full Project permission bundle captured when project-token role assignment was introduced.'
        AND scope_type = 'PROJECT'
        AND subject_type = 'PROJECT_TOKEN'
        AND definition_type = 'BUILT_IN'
        AND status = 'ACTIVE') <> 1 THEN
        RAISE EXCEPTION 'project-token compatibility role metadata mismatch';
    END IF;
    IF (SELECT count(*) FROM public.role_permissions
        WHERE role_definition_id = (
            SELECT id
            FROM public.role_definitions
            WHERE organization_id IS NULL
              AND key = 'project_token_full_access'
              AND definition_type = 'BUILT_IN'
        ) AND scope_type = 'PROJECT' AND status = 'ACTIVE') <> 31 THEN
        RAISE EXCEPTION 'project-token compatibility role permission count mismatch';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM public.role_permissions role_permission
        JOIN public.permission_definitions permission_definition
          ON permission_definition.id = role_permission.permission_definition_id
        JOIN public.role_definitions role_definition
          ON role_definition.id = role_permission.role_definition_id
        WHERE role_definition.organization_id IS NULL
          AND role_definition.key = 'project_token_full_access'
          AND role_definition.definition_type = 'BUILT_IN'
          AND (permission_definition.scope_type <> 'PROJECT' OR permission_definition.status <> 'ACTIVE')
    ) THEN
        RAISE EXCEPTION 'project-token compatibility role permission mismatch';
    END IF;
    IF EXISTS (SELECT 1 FROM public.project_credentials
        WHERE role_definition_id IS NULL OR role_subject_type IS NULL) THEN
        RAISE EXCEPTION 'project credential role backfill incomplete';
    END IF;
END $$;

ALTER TABLE public.project_credentials
    ALTER COLUMN role_definition_id SET NOT NULL,
    ALTER COLUMN role_subject_type SET NOT NULL,
    ADD CONSTRAINT project_credentials_role_subject_check
        CHECK (role_subject_type = 'PROJECT_TOKEN'),
    ADD CONSTRAINT project_credentials_role_definition_fk
        FOREIGN KEY (role_definition_id, role_subject_type)
        REFERENCES public.role_definitions (id, subject_type) ON DELETE RESTRICT;

COMMIT;
