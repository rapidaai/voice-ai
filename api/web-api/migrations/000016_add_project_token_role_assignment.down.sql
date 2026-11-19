BEGIN;

ALTER TABLE public.project_credentials
    DROP CONSTRAINT project_credentials_role_definition_fk,
    DROP CONSTRAINT project_credentials_role_subject_check,
    DROP COLUMN role_subject_type,
    DROP COLUMN role_definition_id;

DELETE FROM public.role_permissions role_permission
USING public.role_definitions role_definition
WHERE role_permission.role_definition_id = role_definition.id
  AND role_definition.organization_id IS NULL
  AND role_definition.key = 'project_token_full_access'
  AND role_definition.definition_type = 'BUILT_IN';

DELETE FROM public.role_definitions
WHERE organization_id IS NULL
  AND key = 'project_token_full_access'
  AND definition_type = 'BUILT_IN';

ALTER TABLE public.role_definitions
    DROP CONSTRAINT role_definitions_subject_reference_unique;

COMMIT;
