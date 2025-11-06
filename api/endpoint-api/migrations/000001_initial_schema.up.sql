
CREATE TABLE public.endpoint_audits (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    endpoint_id bigint NOT NULL,
    endpoint_provider_model_id bigint NOT NULL,
    asset_prefix text,
    time_taken bigint,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    metric json,
    metrics json,
    source character varying DEFAULT 'web-app'::character varying
);


CREATE TABLE public.endpoint_cachings (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    endpoint_id bigint NOT NULL,
    cache_type character varying(200) NOT NULL,
    expiry_interval bigint DEFAULT 0 NOT NULL,
    match_threshold double precision DEFAULT 1.0 NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL
);


CREATE TABLE public.endpoint_log_arguments (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    name character varying(200) NOT NULL,
    value text NOT NULL,
    endpoint_log_id bigint NOT NULL
);



CREATE TABLE public.endpoint_log_metadata (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    endpoint_log_id bigint NOT NULL
);


CREATE TABLE public.endpoint_log_metrics (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    name character varying(200) NOT NULL,
    value text NOT NULL,
    description text,
    endpoint_log_id bigint NOT NULL
);


CREATE TABLE public.endpoint_log_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    endpoint_log_id bigint NOT NULL
);


CREATE TABLE public.endpoint_logs (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    source character varying(50) NOT NULL,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    endpoint_id bigint NOT NULL,
    endpoint_provider_model_id bigint NOT NULL,
    request text,
    response text,
    time_taken bigint
);



CREATE TABLE public.endpoint_provider_model_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    endpoint_provider_model_id bigint NOT NULL
);



CREATE TABLE public.endpoint_provider_models (
    id bigint,
    created_date timestamp without time zone,
    updated_date timestamp without time zone,
    status character varying(50),
    endpoint_id bigint,
    created_by bigint,
    request jsonb,
    model_provider_id bigint,
    description text,
    model_provider_name character varying(200) DEFAULT 'azure-openai'::character varying NOT NULL,
    updated_by bigint
);


CREATE TABLE public.endpoint_retries (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    endpoint_id bigint NOT NULL,
    retry_type character varying(200) NOT NULL,
    max_attempts bigint DEFAULT 0 NOT NULL,
    delay_seconds bigint DEFAULT 0 NOT NULL,
    exponential_backoff boolean DEFAULT true NOT NULL,
    retryables character varying(1000),
    created_by bigint NOT NULL,
    updated_by bigint,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL
);



CREATE TABLE public.endpoint_tags (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    endpoint_id bigint NOT NULL,
    tag character varying(1000),
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL
);



CREATE TABLE public.endpoint_token_audits (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    endpoint_audit_id bigint NOT NULL,
    input_token_count bigint,
    output_token_count bigint,
    input_unit_price double precision,
    output_unit_price double precision
);


CREATE TABLE public.endpoints (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    visibility character varying(50) DEFAULT 'private'::character varying NOT NULL,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    endpoint_provider_model_id bigint,
    source character varying(50),
    source_identifier bigint,
    updated_by bigint,
    created_by bigint,
    retry_enable boolean DEFAULT false NOT NULL,
    cache_enable boolean DEFAULT false NOT NULL,
    name character varying(500),
    description text
);



ALTER TABLE ONLY public.endpoint_audits
    ADD CONSTRAINT endpoint_audits_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_cachings
    ADD CONSTRAINT endpoint_cachings_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_log_arguments
    ADD CONSTRAINT endpoint_log_arguments_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_log_metadata
    ADD CONSTRAINT endpoint_log_metadata_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_log_metrics
    ADD CONSTRAINT endpoint_log_metrics_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_log_options
    ADD CONSTRAINT endpoint_log_options_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_logs
    ADD CONSTRAINT endpoint_logs_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_provider_model_options
    ADD CONSTRAINT endpoint_provider_model_options_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_retries
    ADD CONSTRAINT endpoint_retries_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_tags
    ADD CONSTRAINT endpoint_tags_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_token_audits
    ADD CONSTRAINT endpoint_token_audits_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoints
    ADD CONSTRAINT endpoints_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.endpoint_log_metadata
    ADD CONSTRAINT uk_endpoint_log_id UNIQUE (key, endpoint_log_id);



ALTER TABLE ONLY public.endpoint_log_arguments
    ADD CONSTRAINT uk_endpoint_log_id_mtd UNIQUE (name, endpoint_log_id);



ALTER TABLE ONLY public.endpoint_log_metrics
    ADD CONSTRAINT uk_endpoint_log_id_mtrs UNIQUE (name, endpoint_log_id);



ALTER TABLE ONLY public.endpoint_log_options
    ADD CONSTRAINT uk_endpoint_log_id_opts UNIQUE (key, endpoint_log_id);



ALTER TABLE ONLY public.endpoint_provider_model_options
    ADD CONSTRAINT uk_endpoint_provider_model_id UNIQUE (key, endpoint_provider_model_id);



CREATE UNIQUE INDEX idx_endpoint_cachings ON public.endpoint_cachings USING btree (endpoint_id);



CREATE UNIQUE INDEX idx_endpoint_retries ON public.endpoint_retries USING btree (endpoint_id, retry_type);



CREATE UNIQUE INDEX idx_endpoint_tags ON public.endpoint_tags USING btree (endpoint_id);



CREATE INDEX iea_idx_ea_id ON public.endpoint_token_audits USING btree (endpoint_audit_id);



CREATE INDEX iea_idx_ep_id_epm_id ON public.endpoint_audits USING btree (endpoint_id, endpoint_provider_model_id);
