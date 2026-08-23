SET lock_timeout = '5s';

CREATE TABLE public.service_identities (
    id bigint PRIMARY KEY CHECK (id > 0),
    name character varying(200) NOT NULL UNIQUE,
    status character varying(50) NOT NULL DEFAULT 'ACTIVE',
    signing_key_id character varying(200) NOT NULL,
    signing_public_key text NOT NULL,
    created_actor_type character varying(32) NOT NULL,
    created_actor_id bigint NOT NULL CHECK (created_actor_id > 0),
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    created_date timestamp without time zone NOT NULL DEFAULT now(),
    updated_date timestamp without time zone,
    archived_date timestamp without time zone,
    CONSTRAINT service_identities_created_actor_pair CHECK (
        created_actor_type IN ('user', 'project', 'organization', 'service', 'system')
        AND created_actor_id BETWEEN 1 AND 9223372036854775807
    ),
    CONSTRAINT service_identities_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    )
);

CREATE TABLE public.system_identities (
    id bigint PRIMARY KEY CHECK (id > 0),
    name character varying(200) NOT NULL,
    owning_service_id bigint NOT NULL REFERENCES public.service_identities(id),
    status character varying(50) NOT NULL DEFAULT 'ACTIVE',
    created_actor_type character varying(32) NOT NULL,
    created_actor_id bigint NOT NULL CHECK (created_actor_id > 0),
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    created_date timestamp without time zone NOT NULL DEFAULT now(),
    updated_date timestamp without time zone,
    archived_date timestamp without time zone,
    CONSTRAINT system_identities_owner_name_unique UNIQUE (owning_service_id, name),
    CONSTRAINT system_identities_created_actor_pair CHECK (
        created_actor_type IN ('user', 'project', 'organization', 'service', 'system')
        AND created_actor_id BETWEEN 1 AND 9223372036854775807
    ),
    CONSTRAINT system_identities_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    )
);

RESET lock_timeout;
