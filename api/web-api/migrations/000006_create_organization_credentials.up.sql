CREATE TABLE public.organization_credentials (
    id bigint PRIMARY KEY CHECK (id > 0),
    organization_id bigint NOT NULL CHECK (organization_id > 0),
    name character varying(200) NOT NULL,
    key character varying(200) NOT NULL UNIQUE,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_actor_type character varying(32) NOT NULL,
    created_actor_id bigint NOT NULL CHECK (created_actor_id > 0),
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    archived_date timestamp without time zone,
    CONSTRAINT organization_credentials_status_check CHECK (status IN ('ACTIVE', 'ARCHIEVE')),
    CONSTRAINT organization_credentials_created_actor_type_check CHECK (
        created_actor_type IN ('user', 'project', 'organization', 'service', 'system')
    ),
    CONSTRAINT organization_credentials_updated_actor_check CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL) OR
        (
            updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND
            updated_actor_id > 0
        )
    )
);

CREATE INDEX organization_credentials_organization_id_idx
    ON public.organization_credentials (organization_id);
