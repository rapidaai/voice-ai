CREATE TABLE public.product_usages (
    id bigint PRIMARY KEY CHECK (id > 0),
    organization_id bigint NOT NULL CHECK (organization_id > 0),
    project_id bigint NOT NULL CHECK (project_id > 0),
    usage_id character varying(36) NOT NULL,
    usage_type character varying(100) NOT NULL,
    usages bigint NOT NULL CHECK (usages > 0),
    unit character varying(32) NOT NULL,
    occurred_at timestamp(6) without time zone NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_actor_type character varying(32) NOT NULL,
    created_actor_id bigint NOT NULL CHECK (created_actor_id > 0),
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    CONSTRAINT product_usages_status_check CHECK (status IN ('ACTIVE', 'ARCHIEVE')),
    CONSTRAINT product_usages_created_actor_check CHECK (
        created_actor_type IN ('user', 'project', 'organization', 'service', 'system')
    ),
    CONSTRAINT product_usages_updated_actor_check CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL) OR
        (
            updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND
            updated_actor_id > 0
        )
    ),
    CONSTRAINT product_usages_tenant_usage_id_key UNIQUE (
        organization_id,
        project_id,
        usage_id
    )
);

CREATE INDEX product_usages_project_type_occurred_at_idx
    ON public.product_usages (project_id, usage_type, occurred_at);

CREATE INDEX product_usages_organization_type_occurred_at_idx
    ON public.product_usages (organization_id, usage_type, occurred_at);
