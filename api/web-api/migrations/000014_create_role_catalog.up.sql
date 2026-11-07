BEGIN;

CREATE TABLE public.permission_definitions (
    id bigint PRIMARY KEY CHECK (id > 0),
    key character varying(200) NOT NULL UNIQUE,
    scope_type character varying(32) NOT NULL,
    name character varying(200) NOT NULL,
    description character varying(400) NOT NULL,
    token_eligible boolean NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE' NOT NULL,
    created_actor_type character varying(32) NOT NULL,
    created_actor_id bigint,
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    CONSTRAINT permission_definitions_scope_check CHECK (scope_type IN ('ORGANIZATION', 'PROJECT')),
    CONSTRAINT permission_definitions_token_scope_check CHECK (scope_type = 'PROJECT' OR token_eligible = false),
    CONSTRAINT permission_definitions_reference_unique UNIQUE (id, scope_type, token_eligible),
    CONSTRAINT permission_definitions_status_check CHECK (status IN ('ACTIVE', 'ARCHIEVE')),
    CONSTRAINT permission_definitions_created_actor_check CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL) OR
        (created_actor_type IN ('user', 'project', 'organization', 'service', 'system')
            AND created_actor_id IS NOT NULL AND created_actor_id > 0)
    ),
    CONSTRAINT permission_definitions_updated_actor_check CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL) OR
        (updated_actor_type IS NOT NULL AND updated_actor_id IS NOT NULL
            AND updated_actor_type IN ('user', 'project', 'organization', 'service', 'system')
            AND updated_actor_id > 0)
    )
);

CREATE TABLE public.role_definitions (
    id bigint PRIMARY KEY CHECK (id > 0),
    organization_id bigint,
    key character varying(200) NOT NULL,
    name character varying(200) NOT NULL,
    description character varying(400) NOT NULL,
    scope_type character varying(32) NOT NULL,
    subject_type character varying(32) NOT NULL,
    definition_type character varying(32) NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE' NOT NULL,
    created_actor_type character varying(32) NOT NULL,
    created_actor_id bigint,
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    CONSTRAINT role_definitions_organization_fk FOREIGN KEY (organization_id)
        REFERENCES public.organizations (id) ON DELETE RESTRICT,
    CONSTRAINT role_definitions_scope_check CHECK (scope_type IN ('ORGANIZATION', 'PROJECT')),
    CONSTRAINT role_definitions_subject_check CHECK (subject_type IN ('USER', 'PROJECT_TOKEN')),
    CONSTRAINT role_definitions_type_check CHECK (definition_type IN ('BUILT_IN', 'CUSTOM')),
    CONSTRAINT role_definitions_owner_check CHECK (
        (definition_type = 'BUILT_IN' AND organization_id IS NULL) OR
        (definition_type = 'CUSTOM' AND organization_id IS NOT NULL)
    ),
    CONSTRAINT role_definitions_token_scope_check CHECK (subject_type <> 'PROJECT_TOKEN' OR scope_type = 'PROJECT'),
    CONSTRAINT role_definitions_reference_unique UNIQUE (id, scope_type, subject_type),
    CONSTRAINT role_definitions_status_check CHECK (status IN ('ACTIVE', 'ARCHIEVE')),
    CONSTRAINT role_definitions_created_actor_check CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL) OR
        (created_actor_type IN ('user', 'project', 'organization', 'service', 'system')
            AND created_actor_id IS NOT NULL AND created_actor_id > 0)
    ),
    CONSTRAINT role_definitions_updated_actor_check CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL) OR
        (updated_actor_type IS NOT NULL AND updated_actor_id IS NOT NULL
            AND updated_actor_type IN ('user', 'project', 'organization', 'service', 'system')
            AND updated_actor_id > 0)
    )
);

CREATE UNIQUE INDEX role_definitions_builtin_key_idx
    ON public.role_definitions (key) WHERE definition_type = 'BUILT_IN';
CREATE UNIQUE INDEX role_definitions_custom_key_idx
    ON public.role_definitions (organization_id, key) WHERE definition_type = 'CUSTOM';

CREATE TABLE public.role_permissions (
    id bigint PRIMARY KEY CHECK (id > 0),
    role_definition_id bigint NOT NULL,
    permission_definition_id bigint NOT NULL,
    scope_type character varying(32) NOT NULL,
    subject_type character varying(32) NOT NULL,
    token_eligible boolean NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE' NOT NULL,
    created_actor_type character varying(32) NOT NULL,
    created_actor_id bigint,
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    CONSTRAINT role_permissions_role_fk FOREIGN KEY (role_definition_id, scope_type, subject_type)
        REFERENCES public.role_definitions (id, scope_type, subject_type) ON DELETE RESTRICT,
    CONSTRAINT role_permissions_permission_fk FOREIGN KEY (permission_definition_id, scope_type, token_eligible)
        REFERENCES public.permission_definitions (id, scope_type, token_eligible) ON DELETE RESTRICT,
    CONSTRAINT role_permissions_scope_check CHECK (scope_type IN ('ORGANIZATION', 'PROJECT')),
    CONSTRAINT role_permissions_subject_check CHECK (subject_type IN ('USER', 'PROJECT_TOKEN')),
    CONSTRAINT role_permissions_token_eligibility_check CHECK (subject_type <> 'PROJECT_TOKEN' OR token_eligible = true),
    CONSTRAINT role_permissions_status_check CHECK (status IN ('ACTIVE', 'ARCHIEVE')),
    CONSTRAINT role_permissions_created_actor_check CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL) OR
        (created_actor_type IN ('user', 'project', 'organization', 'service', 'system')
            AND created_actor_id IS NOT NULL AND created_actor_id > 0)
    ),
    CONSTRAINT role_permissions_updated_actor_check CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL) OR
        (updated_actor_type IS NOT NULL AND updated_actor_id IS NOT NULL
            AND updated_actor_type IN ('user', 'project', 'organization', 'service', 'system')
            AND updated_actor_id > 0)
    )
);

CREATE UNIQUE INDEX role_permissions_active_pair_idx
    ON public.role_permissions (role_definition_id, permission_definition_id) WHERE status = 'ACTIVE';

WITH permissions(id, key, scope_type, token_eligible) AS (
    VALUES
        (1, 'organization.read', 'ORGANIZATION', false),
        (2, 'organization.update', 'ORGANIZATION', false),
        (3, 'organization.delete', 'ORGANIZATION', false),
        (4, 'organization.ownership.manage', 'ORGANIZATION', false),
        (5, 'organization.billing.read', 'ORGANIZATION', false),
        (6, 'organization.billing.manage', 'ORGANIZATION', false),
        (7, 'organization.security.read', 'ORGANIZATION', false),
        (8, 'organization.security.manage', 'ORGANIZATION', false),
        (9, 'organization.sessions.revoke', 'ORGANIZATION', false),
        (10, 'organization.members.read', 'ORGANIZATION', false),
        (11, 'organization.members.invite', 'ORGANIZATION', false),
        (12, 'organization.members.remove', 'ORGANIZATION', false),
        (13, 'organization.projects.read', 'ORGANIZATION', false),
        (14, 'organization.projects.create', 'ORGANIZATION', false),
        (15, 'organization.projects.archive', 'ORGANIZATION', false),
        (16, 'organization.roles.read', 'ORGANIZATION', false),
        (17, 'organization.roles.create', 'ORGANIZATION', false),
        (18, 'organization.roles.update', 'ORGANIZATION', false),
        (19, 'organization.roles.archive', 'ORGANIZATION', false),
        (20, 'organization.access.read', 'ORGANIZATION', false),
        (21, 'organization.access.manage', 'ORGANIZATION', false),
        (22, 'organization.project_access_tokens.create', 'ORGANIZATION', false),
        (23, 'organization.project_access_tokens.read_metadata', 'ORGANIZATION', false),
        (24, 'organization.project_access_tokens.reduce_access', 'ORGANIZATION', false),
        (25, 'organization.project_access_tokens.rotate', 'ORGANIZATION', false),
        (26, 'organization.project_access_tokens.revoke', 'ORGANIZATION', false),
        (27, 'organization.provider_credentials.read_metadata', 'ORGANIZATION', false),
        (28, 'organization.provider_credentials.create', 'ORGANIZATION', false),
        (29, 'organization.provider_credentials.update', 'ORGANIZATION', false),
        (30, 'organization.provider_credentials.rotate', 'ORGANIZATION', false),
        (31, 'organization.provider_credentials.revoke', 'ORGANIZATION', false),
        (32, 'organization.audit.read', 'ORGANIZATION', false),
        (33, 'organization.audit.export', 'ORGANIZATION', false),
        (34, 'project.read', 'PROJECT', true),
        (35, 'project.update', 'PROJECT', false),
        (36, 'project.archive', 'PROJECT', false),
        (37, 'project.roles.read', 'PROJECT', false),
        (38, 'project.access.read', 'PROJECT', false),
        (39, 'project.access.manage', 'PROJECT', false),
        (40, 'project.resources.read', 'PROJECT', true),
        (41, 'project.resources.create', 'PROJECT', true),
        (42, 'project.resources.update', 'PROJECT', true),
        (43, 'project.resources.delete', 'PROJECT', true),
        (44, 'project.deployments.read', 'PROJECT', true),
        (45, 'project.deployments.manage', 'PROJECT', true),
        (46, 'project.runtime.invoke', 'PROJECT', true),
        (47, 'project.conversations.read', 'PROJECT', true),
        (48, 'project.conversations.delete', 'PROJECT', true),
        (49, 'project.conversations.export', 'PROJECT', true),
        (50, 'project.observability.read', 'PROJECT', true),
        (51, 'project.observability.export', 'PROJECT', true),
        (52, 'project.access_tokens.create', 'PROJECT', false),
        (53, 'project.access_tokens.read_metadata', 'PROJECT', false),
        (54, 'project.access_tokens.reduce_access', 'PROJECT', false),
        (55, 'project.access_tokens.rotate', 'PROJECT', false),
        (56, 'project.access_tokens.revoke', 'PROJECT', false),
        (57, 'project.provider_credentials.read_metadata', 'PROJECT', true),
        (58, 'project.provider_credentials.create', 'PROJECT', false),
        (59, 'project.provider_credentials.update', 'PROJECT', false),
        (60, 'project.provider_credentials.rotate', 'PROJECT', false),
        (61, 'project.provider_credentials.revoke', 'PROJECT', false),
        (62, 'project.provider_credentials.use', 'PROJECT', true),
        (63, 'project.audit.read', 'PROJECT', true),
        (64, 'project.audit.export', 'PROJECT', true)
)
INSERT INTO public.permission_definitions (
    id, key, scope_type, name, description, token_eligible,
    status, created_actor_type, created_actor_id
)
SELECT
    id, key, scope_type, initcap(translate(key, '._', '  ')), 'Allows ' || key || '.', token_eligible,
    'ACTIVE', 'unknown', NULL
FROM permissions;

WITH roles(id, key, scope_type, subject_type) AS (
    VALUES
        (101, 'organization_owner', 'ORGANIZATION', 'USER'),
        (102, 'organization_admin', 'ORGANIZATION', 'USER'),
        (103, 'access_admin', 'ORGANIZATION', 'USER'),
        (104, 'security_admin', 'ORGANIZATION', 'USER'),
        (105, 'credential_admin', 'ORGANIZATION', 'USER'),
        (106, 'billing_admin', 'ORGANIZATION', 'USER'),
        (107, 'organization_auditor', 'ORGANIZATION', 'USER'),
        (108, 'organization_member', 'ORGANIZATION', 'USER'),
        (109, 'project_admin', 'PROJECT', 'USER'),
        (110, 'project_developer', 'PROJECT', 'USER'),
        (111, 'project_operator', 'PROJECT', 'USER'),
        (112, 'project_viewer', 'PROJECT', 'USER'),
        (113, 'project_auditor', 'PROJECT', 'USER'),
        (114, 'runtime_invoker', 'PROJECT', 'PROJECT_TOKEN'),
        (115, 'automation_client', 'PROJECT', 'PROJECT_TOKEN'),
        (116, 'observer_client', 'PROJECT', 'PROJECT_TOKEN')
)
INSERT INTO public.role_definitions (
    id, organization_id, key, name, description, scope_type, subject_type, definition_type,
    status, created_actor_type, created_actor_id
)
SELECT
    id, NULL, key, initcap(translate(key, '._', '  ')), 'Built-in role ' || key || '.',
    scope_type, subject_type, 'BUILT_IN', 'ACTIVE', 'unknown', NULL
FROM roles;

WITH role_permission_keys(role_key, permission_key) AS (
    SELECT 'organization_owner', key FROM public.permission_definitions WHERE scope_type = 'ORGANIZATION'
    UNION ALL
    SELECT 'organization_admin', key FROM public.permission_definitions
      WHERE scope_type = 'ORGANIZATION' AND key NOT IN (
        'organization.delete', 'organization.ownership.manage', 'organization.billing.read',
        'organization.billing.manage', 'organization.security.read', 'organization.security.manage',
        'organization.sessions.revoke')
    UNION ALL
    SELECT 'access_admin', key FROM public.permission_definitions WHERE key IN (
        'organization.read', 'organization.members.read', 'organization.members.invite',
        'organization.members.remove', 'organization.projects.read', 'organization.roles.read',
        'organization.roles.create', 'organization.roles.update', 'organization.roles.archive',
        'organization.access.read', 'organization.access.manage')
    UNION ALL
    SELECT 'security_admin', key FROM public.permission_definitions WHERE key IN (
        'organization.read', 'organization.security.read', 'organization.security.manage',
        'organization.sessions.revoke', 'organization.access.read', 'organization.audit.read',
        'organization.audit.export')
    UNION ALL
    SELECT 'credential_admin', key FROM public.permission_definitions
      WHERE key = 'organization.read' OR key LIKE 'organization.project_access_tokens.%'
         OR key LIKE 'organization.provider_credentials.%'
    UNION ALL
    SELECT 'billing_admin', key FROM public.permission_definitions WHERE key IN (
        'organization.read', 'organization.billing.read', 'organization.billing.manage')
    UNION ALL
    SELECT 'organization_auditor', key FROM public.permission_definitions WHERE key IN (
        'organization.read', 'organization.billing.read', 'organization.security.read',
        'organization.members.read', 'organization.projects.read', 'organization.roles.read',
        'organization.access.read', 'organization.audit.read', 'organization.audit.export')
    UNION ALL
    SELECT 'organization_member', key FROM public.permission_definitions WHERE key IN (
        'organization.read', 'organization.projects.read')
    UNION ALL
    SELECT 'project_admin', key FROM public.permission_definitions WHERE scope_type = 'PROJECT'
    UNION ALL
    SELECT 'project_developer', key FROM public.permission_definitions WHERE key IN (
        'project.read', 'project.resources.read', 'project.resources.create', 'project.resources.update',
        'project.resources.delete', 'project.deployments.read', 'project.runtime.invoke',
        'project.conversations.read', 'project.observability.read',
        'project.provider_credentials.read_metadata', 'project.provider_credentials.use')
    UNION ALL
    SELECT 'project_operator', key FROM public.permission_definitions WHERE key IN (
        'project.read', 'project.resources.read', 'project.deployments.read',
        'project.deployments.manage', 'project.runtime.invoke', 'project.conversations.read',
        'project.observability.read', 'project.observability.export', 'project.audit.read')
    UNION ALL
    SELECT 'project_viewer', key FROM public.permission_definitions WHERE key IN (
        'project.read', 'project.resources.read', 'project.deployments.read',
        'project.conversations.read', 'project.observability.read')
    UNION ALL
    SELECT 'project_auditor', key FROM public.permission_definitions WHERE key IN (
        'project.read', 'project.roles.read', 'project.access.read', 'project.deployments.read',
        'project.observability.read', 'project.observability.export',
        'project.audit.read', 'project.audit.export')
    UNION ALL
    SELECT 'runtime_invoker', key FROM public.permission_definitions WHERE key = 'project.runtime.invoke'
    UNION ALL
    SELECT 'automation_client', key FROM public.permission_definitions WHERE key IN (
        'project.read', 'project.resources.read', 'project.resources.create', 'project.resources.update',
        'project.resources.delete', 'project.deployments.read', 'project.deployments.manage',
        'project.runtime.invoke', 'project.observability.read',
        'project.provider_credentials.read_metadata', 'project.provider_credentials.use')
    UNION ALL
    SELECT 'observer_client', key FROM public.permission_definitions WHERE key IN (
        'project.read', 'project.resources.read', 'project.deployments.read',
        'project.conversations.read', 'project.observability.read', 'project.observability.export',
        'project.audit.read', 'project.audit.export')
), resolved AS (
    SELECT
        role_definition.id AS role_id,
        permission_definition.id AS permission_id,
        role_definition.scope_type,
        role_definition.subject_type,
        permission_definition.token_eligible,
        role_definition.key AS role_key,
        permission_definition.key AS permission_key
    FROM role_permission_keys
    JOIN public.role_definitions role_definition ON role_definition.key = role_permission_keys.role_key
    JOIN public.permission_definitions permission_definition ON permission_definition.key = role_permission_keys.permission_key
)
INSERT INTO public.role_permissions (
    id, role_definition_id, permission_definition_id, scope_type, subject_type, token_eligible,
    status, created_actor_type, created_actor_id
)
SELECT
    1000 + row_number() OVER (ORDER BY role_key, permission_key),
    role_id, permission_id, scope_type, subject_type, token_eligible,
    'ACTIVE', 'unknown', NULL
FROM resolved;

DO $$
DECLARE
    invalid_count bigint;
BEGIN
    IF (SELECT count(*) FROM public.permission_definitions) <> 64 THEN
        RAISE EXCEPTION 'permission seed count mismatch';
    END IF;
    IF (SELECT count(*) FROM public.permission_definitions WHERE scope_type = 'ORGANIZATION') <> 33 THEN
        RAISE EXCEPTION 'organization permission seed count mismatch';
    END IF;
    IF (SELECT count(*) FROM public.permission_definitions WHERE scope_type = 'PROJECT') <> 31 THEN
        RAISE EXCEPTION 'project permission seed count mismatch';
    END IF;
    IF (SELECT count(*) FROM public.permission_definitions WHERE token_eligible) <> 17 THEN
        RAISE EXCEPTION 'token-eligible permission seed count mismatch';
    END IF;
    IF (SELECT count(*) FROM public.role_definitions) <> 16 THEN
        RAISE EXCEPTION 'role seed count mismatch';
    END IF;
    IF (SELECT count(*) FROM public.role_permissions) <> 186 THEN
        RAISE EXCEPTION 'role-permission seed count mismatch';
    END IF;

    SELECT count(*) INTO invalid_count
    FROM (
        VALUES
            ('organization_owner', 33), ('organization_admin', 26), ('access_admin', 11),
            ('security_admin', 7), ('credential_admin', 11), ('billing_admin', 3),
            ('organization_auditor', 9), ('organization_member', 2), ('project_admin', 31),
            ('project_developer', 11), ('project_operator', 9), ('project_viewer', 5),
            ('project_auditor', 8), ('runtime_invoker', 1), ('automation_client', 11),
            ('observer_client', 8)
    ) expected(role_key, permission_count)
    LEFT JOIN (
        SELECT role_definition.key AS role_key, count(*) AS permission_count
        FROM public.role_permissions role_permission
        JOIN public.role_definitions role_definition ON role_definition.id = role_permission.role_definition_id
        WHERE role_permission.status = 'ACTIVE'
        GROUP BY role_definition.key
    ) actual USING (role_key)
    WHERE actual.permission_count IS DISTINCT FROM expected.permission_count;

    IF invalid_count <> 0 THEN
        RAISE EXCEPTION 'role bundle seed mismatch';
    END IF;
END $$;

COMMIT;
