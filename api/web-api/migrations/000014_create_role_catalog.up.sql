BEGIN;

CREATE TABLE public.permission_definitions (
    id bigint PRIMARY KEY CHECK (id > 0),
    key character varying(200) NOT NULL UNIQUE,
    scope_type character varying(32) NOT NULL,
    name character varying(200) NOT NULL,
    description character varying(400) NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE' NOT NULL,
    created_actor_type character varying(32) NOT NULL,
    created_actor_id bigint,
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    CONSTRAINT permission_definitions_scope_check CHECK (scope_type IN ('ORGANIZATION', 'PROJECT')),
    CONSTRAINT permission_definitions_reference_unique UNIQUE (id, scope_type),
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
    CONSTRAINT role_definitions_reference_unique UNIQUE (id, scope_type),
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
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE' NOT NULL,
    created_actor_type character varying(32) NOT NULL,
    created_actor_id bigint,
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    CONSTRAINT role_permissions_role_fk FOREIGN KEY (role_definition_id, scope_type)
        REFERENCES public.role_definitions (id, scope_type) ON DELETE RESTRICT,
    CONSTRAINT role_permissions_permission_fk FOREIGN KEY (permission_definition_id, scope_type)
        REFERENCES public.permission_definitions (id, scope_type) ON DELETE RESTRICT,
    CONSTRAINT role_permissions_scope_check CHECK (scope_type IN ('ORGANIZATION', 'PROJECT')),
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

WITH permissions(id, key, scope_type) AS (
    VALUES
        (1, 'organization.read', 'ORGANIZATION'),
        (2, 'organization.update', 'ORGANIZATION'),
        (3, 'organization.delete', 'ORGANIZATION'),
        (4, 'organization.ownership.manage', 'ORGANIZATION'),
        (5, 'organization.billing.read', 'ORGANIZATION'),
        (6, 'organization.billing.manage', 'ORGANIZATION'),
        (7, 'organization.security.read', 'ORGANIZATION'),
        (8, 'organization.security.manage', 'ORGANIZATION'),
        (9, 'organization.sessions.revoke', 'ORGANIZATION'),
        (10, 'organization.members.read', 'ORGANIZATION'),
        (11, 'organization.members.invite', 'ORGANIZATION'),
        (12, 'organization.members.remove', 'ORGANIZATION'),
        (13, 'organization.projects.read', 'ORGANIZATION'),
        (14, 'organization.projects.create', 'ORGANIZATION'),
        (15, 'organization.projects.archive', 'ORGANIZATION'),
        (16, 'organization.roles.read', 'ORGANIZATION'),
        (17, 'organization.roles.create', 'ORGANIZATION'),
        (18, 'organization.roles.update', 'ORGANIZATION'),
        (19, 'organization.roles.archive', 'ORGANIZATION'),
        (20, 'organization.access.read', 'ORGANIZATION'),
        (21, 'organization.access.manage', 'ORGANIZATION'),
        (22, 'organization.project_access_tokens.create', 'ORGANIZATION'),
        (23, 'organization.project_access_tokens.read_metadata', 'ORGANIZATION'),
        (24, 'organization.project_access_tokens.reduce_access', 'ORGANIZATION'),
        (25, 'organization.project_access_tokens.rotate', 'ORGANIZATION'),
        (26, 'organization.project_access_tokens.revoke', 'ORGANIZATION'),
        (27, 'organization.provider_credentials.read_metadata', 'ORGANIZATION'),
        (28, 'organization.provider_credentials.create', 'ORGANIZATION'),
        (29, 'organization.provider_credentials.update', 'ORGANIZATION'),
        (30, 'organization.provider_credentials.rotate', 'ORGANIZATION'),
        (31, 'organization.provider_credentials.revoke', 'ORGANIZATION'),
        (32, 'organization.audit.read', 'ORGANIZATION'),
        (33, 'organization.audit.export', 'ORGANIZATION'),
        (34, 'project.read', 'PROJECT'),
        (35, 'project.update', 'PROJECT'),
        (36, 'project.archive', 'PROJECT'),
        (37, 'project.roles.read', 'PROJECT'),
        (38, 'project.access.read', 'PROJECT'),
        (39, 'project.access.manage', 'PROJECT'),
        (40, 'project.resources.read', 'PROJECT'),
        (41, 'project.resources.create', 'PROJECT'),
        (42, 'project.resources.update', 'PROJECT'),
        (43, 'project.resources.delete', 'PROJECT'),
        (44, 'project.deployments.read', 'PROJECT'),
        (45, 'project.deployments.manage', 'PROJECT'),
        (46, 'project.runtime.invoke', 'PROJECT'),
        (47, 'project.conversations.read', 'PROJECT'),
        (48, 'project.conversations.delete', 'PROJECT'),
        (49, 'project.conversations.export', 'PROJECT'),
        (50, 'project.observability.read', 'PROJECT'),
        (51, 'project.observability.export', 'PROJECT'),
        (52, 'project.access_tokens.create', 'PROJECT'),
        (53, 'project.access_tokens.read_metadata', 'PROJECT'),
        (54, 'project.access_tokens.reduce_access', 'PROJECT'),
        (55, 'project.access_tokens.rotate', 'PROJECT'),
        (56, 'project.access_tokens.revoke', 'PROJECT'),
        (57, 'project.provider_credentials.read_metadata', 'PROJECT'),
        (58, 'project.provider_credentials.create', 'PROJECT'),
        (59, 'project.provider_credentials.update', 'PROJECT'),
        (60, 'project.provider_credentials.rotate', 'PROJECT'),
        (61, 'project.provider_credentials.revoke', 'PROJECT'),
        (62, 'project.provider_credentials.use', 'PROJECT'),
        (63, 'project.audit.read', 'PROJECT'),
        (64, 'project.audit.export', 'PROJECT')
)
INSERT INTO public.permission_definitions (
    id, key, scope_type, name, description,
    status, created_actor_type, created_actor_id
)
SELECT
    id, key, scope_type, initcap(translate(key, '._', '  ')), 'Allows ' || key || '.',
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
        role_definition.key AS role_key,
        permission_definition.key AS permission_key
    FROM role_permission_keys
    JOIN public.role_definitions role_definition ON role_definition.key = role_permission_keys.role_key
    JOIN public.permission_definitions permission_definition ON permission_definition.key = role_permission_keys.permission_key
)
INSERT INTO public.role_permissions (
    id, role_definition_id, permission_definition_id, scope_type,
    status, created_actor_type, created_actor_id
)
SELECT
    1000 + row_number() OVER (ORDER BY role_key, permission_key),
    role_id, permission_id, scope_type,
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
