
CREATE TABLE public.assistant_analyses (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    assistant_id bigint NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    name character varying(200) NOT NULL,
    description text NOT NULL,
    endpoint_id bigint NOT NULL,
    endpoint_version character varying(200) NOT NULL,
    endpoint_parameters text NOT NULL,
    execution_priority bigint NOT NULL
);

CREATE TABLE public.assistant_api_deployments (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    assistant_id bigint NOT NULL,
    name character varying(50) NOT NULL,
    greeting character varying(250),
    mistake character varying(250)
);

CREATE TABLE public.assistant_conversation_messages (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    request jsonb NOT NULL,
    response jsonb,
    source character varying(50) DEFAULT 'web-app'::character varying NOT NULL,
    status character varying(50) DEFAULT 'IN_PROGRESS'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    assistant_id bigint,
    assistant_provider_model_id bigint,
    message_id character varying
);

CREATE TABLE public.assistant_conversations (
    id bigint NOT NULL,
    identifier text NOT NULL,
    assistant_id bigint NOT NULL,
    assistant_provider_model_id bigint NOT NULL,
    name text,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    source character varying(50) DEFAULT 'web-app'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'active'::character varying,
    direction character varying(20) DEFAULT 'inbound'::character varying NOT NULL
);

CREATE TABLE public.assistant_conversation_action_metrics (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    assistant_conversation_action_id bigint NOT NULL,
    assistant_conversation_message_id character varying(50) NOT NULL,
    name character varying(200) NOT NULL,
    value text NOT NULL,
    description text,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone
);
CREATE TABLE public.assistant_conversation_actions (
    id bigint NOT NULL,
    assistant_conversation_message_id character varying NOT NULL,
    external_id character varying NOT NULL,
    action_type character varying NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    request json,
    response json,
    metrics json,
    status character varying,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone
);

CREATE TABLE public.assistant_conversation_arguments (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    name character varying(200) NOT NULL,
    value text NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone
);
CREATE TABLE public.assistant_conversation_contexts (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    result json,
    query json,
    metadata json,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    context_id character varying NOT NULL,
    message_id character varying
);
CREATE TABLE public.assistant_conversation_message_metadata (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    assistant_conversation_message_id character varying(50) NOT NULL,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone
);
CREATE TABLE public.assistant_conversation_message_metrics (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    assistant_conversation_message_id character varying(50) NOT NULL,
    name character varying(200) NOT NULL,
    value text NOT NULL,
    description text,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone
);

CREATE TABLE public.assistant_conversation_message_stages (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    assistant_conversation_message_id character varying(50) NOT NULL,
    stage character varying(255) NOT NULL,
    stage_name character varying(255) NOT NULL,
    lifecycle_id character varying(255) NOT NULL,
    start_timestamp timestamp without time zone,
    end_timestamp timestamp without time zone,
    time_taken bigint,
    additional_data text,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone
);
CREATE TABLE public.assistant_conversation_metadata (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone
);
CREATE TABLE public.assistant_conversation_metrics (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    name character varying(200) NOT NULL,
    value text NOT NULL,
    description text,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone
);

CREATE TABLE public.assistant_conversation_options (
    id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone
);
CREATE TABLE public.assistant_conversation_recordings (
    id bigint NOT NULL,
    created_date timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp with time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint,
    updated_by bigint,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    assistant_id bigint,
    assistant_conversation_id bigint,
    recording_url character varying(200) NOT NULL
);

CREATE TABLE public.assistant_debugger_deployments (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    assistant_id bigint NOT NULL,
    name character varying(50),
    role character varying(50),
    tone character varying(50),
    experties character varying(250),
    greeting character varying(250),
    mistake character varying(250),
    icon character varying(50) NOT NULL
);

CREATE TABLE public.assistant_deployment_audio_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_deployment_audio_id bigint NOT NULL
);
CREATE TABLE public.assistant_deployment_audios (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    assistant_deployment_id bigint NOT NULL,
    audio_type character varying(50) NOT NULL,
    audio_provider_id bigint NOT NULL,
    audio_provider character varying(255) NOT NULL
);
CREATE TABLE public.assistant_deployment_telephony_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_deployment_telephony_id bigint NOT NULL
);

CREATE TABLE public.assistant_deployment_whatsapp_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_deployment_whatsapp_id bigint NOT NULL
);

CREATE TABLE public.assistant_knowledge_logs (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    organization_id bigint NOT NULL,
    knowledge_id bigint NOT NULL,
    assistant_id bigint NOT NULL,
    assistant_conversation_id bigint NOT NULL,
    assistant_conversation_message_id character varying(255) NOT NULL,
    asset_prefix character varying(200) NOT NULL,
    time_taken bigint,
    source character varying(50) NOT NULL
);
CREATE TABLE public.assistant_knowledge_reranker_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_knowledge_id bigint NOT NULL
);
CREATE TABLE public.assistant_knowledges (
    id bigint NOT NULL,
    assistant_id bigint NOT NULL,
    knowledge_id bigint NOT NULL,
    reranker_enable boolean DEFAULT false,
    reranker_provider_model_id bigint,
    top_k bigint,
    score_threshold double precision,
    created_by bigint NOT NULL,
    updated_by bigint,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    retrieval_method character varying(50),
    reranker_provider_id bigint,
    reranker_model_provider_name character varying(200),
    reranker_model_provider_id bigint
);

CREATE TABLE public.assistant_phone_deployments (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    assistant_id bigint NOT NULL,
    name character varying(50) NOT NULL,
    greeting character varying(250) NOT NULL,
    mistake character varying(250) NOT NULL,
    telephony_provider_id bigint NOT NULL,
    telephony_provider character varying(50) NOT NULL
);
CREATE TABLE public.assistant_provider_agentkits (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    assistant_id bigint NOT NULL,
    description character varying(400) NOT NULL,
    url character varying(200) NOT NULL,
    certificate text,
    metadata text
);
CREATE TABLE public.assistant_provider_model_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_provider_model_id bigint NOT NULL
);
CREATE TABLE public.assistant_provider_models (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    assistant_id bigint,
    created_by bigint NOT NULL,
    description text,
    model_provider_id bigint,
    template jsonb,
    model_provider_name character varying(200) DEFAULT 'azure-openai'::character varying NOT NULL
);

CREATE TABLE public.assistant_provider_websockets (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    assistant_id bigint NOT NULL,
    description character varying(400) NOT NULL,
    url character varying(200) NOT NULL,
    headers text,
    parameters text
);

CREATE TABLE public.assistant_tags (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    assistant_id bigint NOT NULL,
    tag character varying(1000),
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);
CREATE TABLE public.assistant_tool_logs (
    id bigint NOT NULL,
    created_date timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp with time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint,
    updated_by bigint,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    assistant_tool_id bigint NOT NULL,
    assistant_tool_name character varying(255) NOT NULL,
    assistant_id bigint,
    assistant_conversation_id bigint,
    assistant_conversation_message_id character varying(255) NOT NULL,
    execution_method character varying(20),
    asset_prefix character varying(200) NOT NULL,
    time_taken bigint
);
CREATE TABLE public.assistant_tool_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    assistant_tool_id bigint NOT NULL
);

CREATE TABLE public.assistant_tools (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    assistant_id bigint NOT NULL,
    name character varying(50) NOT NULL,
    description character varying(400) NOT NULL,
    fields text NOT NULL,
    execution_method character varying(200) NOT NULL
);
CREATE TABLE public.assistant_web_plugin_deployments (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    assistant_id bigint NOT NULL,
    name character varying(50) NOT NULL,
    greeting character varying(200) NOT NULL,
    mistake character varying(200) NOT NULL,
    icon text NOT NULL,
    suggestions text,
    help_center_enabled boolean DEFAULT false NOT NULL,
    product_catalog_enabled boolean DEFAULT false NOT NULL,
    article_catalog_enabled boolean DEFAULT false NOT NULL
);

CREATE TABLE public.assistant_webhook_logs (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    webhook_id bigint NOT NULL,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    assistant_id bigint,
    assistant_conversation_id bigint,
    http_method character varying(200) NOT NULL,
    http_url character varying(400) NOT NULL,
    asset_prefix character varying(200) NOT NULL,
    event character varying(200) NOT NULL,
    response_status bigint,
    time_taken bigint,
    retry_count bigint
);

CREATE TABLE public.assistant_webhooks (
    id bigint NOT NULL,
    assistant_id bigint NOT NULL,
    assistant_events text NOT NULL,
    description text,
    http_method text,
    http_url text,
    http_headers text DEFAULT '{}'::text,
    retry_status_codes text NOT NULL,
    max_retry_count integer,
    timeout_seconds integer,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    execution_priority bigint DEFAULT '-1'::integer,
    http_body text DEFAULT '{}'::text
);

CREATE TABLE public.assistant_whatsapp_deployments (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    assistant_id bigint NOT NULL,
    name character varying(50) NOT NULL,
    role character varying(50) NOT NULL,
    tone character varying(50) NOT NULL,
    experties character varying(250) NOT NULL,
    greeting character varying(250) NOT NULL,
    mistake character varying(250) NOT NULL,
    ending character varying(250) NOT NULL,
    whatsapp_provider_id bigint NOT NULL,
    whatsapp_provider character varying(50) NOT NULL
);
CREATE TABLE public.assistants (
    id bigint NOT NULL,
    name character varying(250),
    description character varying(2000),
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    visibility character varying(50) DEFAULT 'private'::character varying NOT NULL,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    assistant_provider_id bigint,
    language character varying(50) DEFAULT 'english'::character varying NOT NULL,
    source character varying(50),
    source_identifier bigint,
    created_by bigint,
    updated_by bigint,
    assistant_provider character varying(50) DEFAULT 'PROVIDER_MODEL'::character varying NOT NULL
);
CREATE TABLE public.knowledge_collections (
    id bigint NOT NULL,
    knowledge_id bigint NOT NULL,
    provider_model_id bigint NOT NULL,
    provider_id bigint NOT NULL,
    created_date timestamp without time zone NOT NULL,
    updated_date timestamp without time zone,
    name character varying(250) NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    router_provider_id bigint NOT NULL,
    router_provider_model_id bigint NOT NULL
);

CREATE SEQUENCE public.knowledge_document_embeddings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.knowledge_document_embeddings (
    id integer DEFAULT nextval('public.knowledge_document_embeddings_id_seq'::regclass) NOT NULL,
    hash character varying(64) NOT NULL,
    embedding bytea NOT NULL,
    embedding_provider_model_id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    base64 text
);
CREATE TABLE public.knowledge_document_process_rules (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    created_by bigint NOT NULL,
    updated_by bigint,
    knowledge_document_id bigint NOT NULL,
    mode character varying(255) DEFAULT 'automatic'::character varying NOT NULL,
    rules text NOT NULL
);

CREATE SEQUENCE public.knowledge_document_segments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
CREATE TABLE public.knowledge_document_segments (
    id bigint DEFAULT nextval('public.knowledge_document_segments_id_seq'::regclass) NOT NULL,
    knowledge_document_id bigint NOT NULL,
    "position" integer NOT NULL,
    content text NOT NULL,
    answer text,
    word_count integer NOT NULL,
    token_count integer NOT NULL,
    hit_count integer DEFAULT 0 NOT NULL,
    keywords jsonb,
    index_node_id character varying(255),
    index_node_hash character varying(255),
    enabled boolean DEFAULT true NOT NULL,
    disabled_at timestamp without time zone,
    disabled_by uuid,
    status character varying(255) DEFAULT 'waiting'::character varying NOT NULL,
    indexing_at timestamp without time zone,
    completed_at timestamp without time zone,
    error text,
    stopped_at timestamp without time zone,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    knowledge_id bigint NOT NULL
);

CREATE TABLE public.knowledge_documents (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    created_by bigint NOT NULL,
    updated_by bigint,
    knowledge_id bigint NOT NULL,
    language character varying(50) DEFAULT 'english'::character varying NOT NULL,
    name character varying(255),
    description character varying(255),
    document_source text NOT NULL,
    document_path character varying(500),
    index_status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    retrieval_count bigint DEFAULT 0,
    token_count bigint DEFAULT 0,
    word_count bigint DEFAULT 0,
    index_struct text,
    cleaning_completed_at timestamp without time zone,
    splitting_completed_at timestamp without time zone,
    indexing_latency double precision,
    completed_at timestamp without time zone,
    error text,
    parsing_completed_at timestamp without time zone,
    processing_started_at timestamp without time zone,
    stopped_at timestamp without time zone,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    document_size bigint DEFAULT 0,
    document_structure character varying(50)
);
CREATE TABLE public.knowledge_embedding_model_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    knowledge_id bigint NOT NULL
);
CREATE TABLE public.knowledge_logs (
    id bigint NOT NULL,
    created_date timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp with time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint,
    updated_by bigint,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    knowledge_id bigint NOT NULL,
    retrieval_method character varying(50),
    top_k integer,
    score_threshold real,
    document_count integer,
    asset_prefix character varying(200) NOT NULL,
    time_taken bigint,
    additional_data text
);
CREATE TABLE public.knowledge_model_options (
    id bigint NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_date timestamp without time zone,
    key character varying(200) NOT NULL,
    value text NOT NULL,
    knowledge_id bigint NOT NULL
);
CREATE TABLE public.knowledge_tags (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    knowledge_id bigint NOT NULL,
    tag character varying(1000),
    created_by bigint NOT NULL,
    updated_by bigint
);
CREATE TABLE public.knowledges (
    id bigint NOT NULL,
    name character varying NOT NULL,
    description text,
    visibility character varying DEFAULT 'private'::character varying NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    embedding_model_provider_name character varying(200) NOT NULL,
    embedding_model_provider_id bigint,
    storage_namespace character varying(400)
);
CREATE TABLE public.tools (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint,
    icon character varying(250) NOT NULL,
    name character varying(250) NOT NULL,
    description character varying(500) NOT NULL,
    code character varying(250) NOT NULL,
    setup_options text NOT NULL,
    initialize_options text NOT NULL,
    visibility character varying(100) DEFAULT 'PUBLIC'::character varying NOT NULL,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL
);

ALTER TABLE ONLY public.assistant_analyses
    ADD CONSTRAINT assistant_analyses_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_api_deployments
    ADD CONSTRAINT assistant_api_deployments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_messages
    ADD CONSTRAINT assistant_conversaction_messages_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversations
    ADD CONSTRAINT assistant_conversactions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_action_metrics
    ADD CONSTRAINT assistant_conversation_action_metrics_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_actions
    ADD CONSTRAINT assistant_conversation_actions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_arguments
    ADD CONSTRAINT assistant_conversation_arguments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_contexts
    ADD CONSTRAINT assistant_conversation_contexts_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_message_metadata
    ADD CONSTRAINT assistant_conversation_message_metadata_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_message_metrics
    ADD CONSTRAINT assistant_conversation_message_metrics_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_message_stages
    ADD CONSTRAINT assistant_conversation_message_stages_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_metadata
    ADD CONSTRAINT assistant_conversation_metadata_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_metrics
    ADD CONSTRAINT assistant_conversation_metrics_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_options
    ADD CONSTRAINT assistant_conversation_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_recordings
    ADD CONSTRAINT assistant_conversation_recordings_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_debugger_deployments
    ADD CONSTRAINT assistant_debugger_deployments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_deployment_audio_options
    ADD CONSTRAINT assistant_deployment_audio_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_deployment_audios
    ADD CONSTRAINT assistant_deployment_audios_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_deployment_telephony_options
    ADD CONSTRAINT assistant_deployment_telephony_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_deployment_whatsapp_options
    ADD CONSTRAINT assistant_deployment_whatsapp_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_knowledge_logs
    ADD CONSTRAINT assistant_knowledge_logs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_knowledge_reranker_options
    ADD CONSTRAINT assistant_knowledge_reranker_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_phone_deployments
    ADD CONSTRAINT assistant_phone_deployments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_provider_agentkits
    ADD CONSTRAINT assistant_provider_agentkits_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_provider_model_options
    ADD CONSTRAINT assistant_provider_model_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_provider_websockets
    ADD CONSTRAINT assistant_provider_websockets_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_tool_logs
    ADD CONSTRAINT assistant_tool_logs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_tool_options
    ADD CONSTRAINT assistant_tool_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_tools
    ADD CONSTRAINT assistant_tools_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_web_plugin_deployments
    ADD CONSTRAINT assistant_web_plugin_deployments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_webhook_logs
    ADD CONSTRAINT assistant_webhook_logs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_webhooks
    ADD CONSTRAINT assistant_webhooks_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_whatsapp_deployments
    ADD CONSTRAINT assistant_whatsapp_deployments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_contexts
    ADD CONSTRAINT ctx_idx_assistant_conversation_contexts UNIQUE (assistant_conversation_id, context_id);
ALTER TABLE ONLY public.knowledge_collections
    ADD CONSTRAINT knowledge_collections_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.knowledge_document_embeddings
    ADD CONSTRAINT knowledge_document_embeddings_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.knowledge_document_process_rules
    ADD CONSTRAINT knowledge_document_process_rules_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.knowledge_document_segments
    ADD CONSTRAINT knowledge_document_segments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.knowledge_embedding_model_options
    ADD CONSTRAINT knowledge_embedding_model_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.knowledge_logs
    ADD CONSTRAINT knowledge_logs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.knowledge_model_options
    ADD CONSTRAINT knowledge_model_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.knowledges
    ADD CONSTRAINT knowledges_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tools
    ADD CONSTRAINT tools_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.assistant_conversation_arguments
    ADD CONSTRAINT uk_an_arguments UNIQUE (assistant_conversation_id, name);
ALTER TABLE ONLY public.assistant_conversation_metadata
    ADD CONSTRAINT uk_an_metadata UNIQUE (assistant_conversation_id, key);
ALTER TABLE ONLY public.assistant_conversation_options
    ADD CONSTRAINT uk_an_options UNIQUE (assistant_conversation_id, key);
ALTER TABLE ONLY public.assistant_conversation_action_metrics
    ADD CONSTRAINT uk_assistant_conversation_action_metrics UNIQUE (assistant_conversation_action_id, name);
ALTER TABLE ONLY public.assistant_conversation_message_metadata
    ADD CONSTRAINT uk_assistant_conversation_message_metadata UNIQUE (assistant_conversation_message_id, key);
ALTER TABLE ONLY public.assistant_conversation_message_metrics
    ADD CONSTRAINT uk_assistant_conversation_message_metrics UNIQUE (assistant_conversation_message_id, name);
ALTER TABLE ONLY public.assistant_conversation_message_stages
    ADD CONSTRAINT uk_assistant_conversation_message_stages UNIQUE (assistant_conversation_message_id, stage);
ALTER TABLE ONLY public.assistant_conversation_metrics
    ADD CONSTRAINT uk_assistant_conversation_name UNIQUE (assistant_conversation_id, name);
ALTER TABLE ONLY public.assistant_deployment_audio_options
    ADD CONSTRAINT uk_assistant_deployment_audio_option UNIQUE (key, assistant_deployment_audio_id);
ALTER TABLE ONLY public.assistant_deployment_telephony_options
    ADD CONSTRAINT uk_assistant_deployment_telephony_options UNIQUE (key, assistant_deployment_telephony_id);
ALTER TABLE ONLY public.assistant_deployment_whatsapp_options
    ADD CONSTRAINT uk_assistant_deployment_whatsapp_options UNIQUE (key, assistant_deployment_whatsapp_id);
ALTER TABLE ONLY public.assistant_knowledge_reranker_options
    ADD CONSTRAINT uk_assistant_knowledge_configuration_id UNIQUE (key, assistant_knowledge_id);
ALTER TABLE ONLY public.assistant_provider_model_options
    ADD CONSTRAINT uk_assistant_provider_model_id UNIQUE (key, assistant_provider_model_id);
ALTER TABLE ONLY public.assistant_tool_options
    ADD CONSTRAINT uk_assistant_tool_id UNIQUE (key, assistant_tool_id);
ALTER TABLE ONLY public.knowledge_embedding_model_options
    ADD CONSTRAINT uk_knowledge_embedding_id UNIQUE (key, knowledge_id);
ALTER TABLE ONLY public.knowledge_model_options
    ADD CONSTRAINT uk_knowledge_id UNIQUE (key, knowledge_id);
ALTER TABLE ONLY public.assistant_knowledges
    ADD CONSTRAINT unique_assistant_knowledge UNIQUE (assistant_id, knowledge_id);
ALTER TABLE ONLY public.assistant_conversation_messages
    ADD CONSTRAINT unique_message_id_assistant_conversation_id UNIQUE (message_id, assistant_conversation_id);

CREATE INDEX assistant_conversation_id_assistant_conversation_contexts ON public.assistant_conversation_contexts USING btree (assistant_conversation_id);
CREATE INDEX idx_assistant_api_deployments_on_assistant_id ON public.assistant_api_deployments USING btree (assistant_id);
CREATE INDEX idx_assistant_conversactions_identifier_assistant_org_proj ON public.assistant_conversations USING btree (identifier, assistant_id, organization_id, project_id);
CREATE INDEX idx_assistant_conversation_action_metrics ON public.assistant_conversation_action_metrics USING btree (assistant_conversation_action_id);
CREATE INDEX idx_assistant_conversation_message_id ON public.assistant_conversation_message_metrics USING btree (assistant_conversation_message_id);
CREATE INDEX idx_assistant_conversation_message_metadata ON public.assistant_conversation_message_metadata USING btree (assistant_conversation_message_id);
CREATE INDEX idx_assistant_conversation_message_stages ON public.assistant_conversation_message_stages USING btree (assistant_conversation_message_id);
CREATE INDEX idx_assistant_debugger_deployments_on_assistant_id ON public.assistant_debugger_deployments USING btree (assistant_id);
CREATE INDEX idx_assistant_deployment_audios_on_deployment_id_and_audio_type ON public.assistant_deployment_audios USING btree (assistant_deployment_id, audio_type);
CREATE INDEX idx_assistant_knowledge_configurations_assistant_id ON public.assistant_knowledges USING btree (assistant_id);
CREATE INDEX idx_assistant_knowledge_configurations_assistant_id_status ON public.assistant_knowledges USING btree (assistant_id, status);
CREATE INDEX idx_assistant_knowledge_configurations_status ON public.assistant_knowledges USING btree (status);
CREATE INDEX idx_assistant_knowledges_on_assistant_id_and_status ON public.assistant_knowledges USING btree (assistant_id, status);
CREATE INDEX idx_assistant_phone_deployments_on_assistant_id ON public.assistant_phone_deployments USING btree (assistant_id);
CREATE INDEX idx_assistant_provider_models_assistant_id ON public.assistant_provider_models USING btree (assistant_id);
CREATE UNIQUE INDEX idx_assistant_provider_models_id ON public.assistant_provider_models USING btree (id);
CREATE INDEX idx_assistant_provider_models_id_assistant_id ON public.assistant_provider_models USING btree (id, assistant_id);
CREATE INDEX idx_assistant_provider_models_on_assistant_id ON public.assistant_provider_models USING btree (assistant_id);
CREATE UNIQUE INDEX idx_assistant_tags ON public.assistant_tags USING btree (assistant_id);
CREATE INDEX idx_assistant_tags_assistant_id ON public.assistant_tags USING btree (assistant_id);
CREATE INDEX idx_assistant_tags_on_assistant_id ON public.assistant_tags USING btree (assistant_id);
CREATE INDEX idx_assistant_tools_on_assistant_id_and_status ON public.assistant_tools USING btree (assistant_id, status);
CREATE INDEX idx_assistant_web_plugin_deployments_on_assistant_id ON public.assistant_web_plugin_deployments USING btree (assistant_id);
CREATE INDEX idx_assistant_whatsapp_deployments_on_assistant_id ON public.assistant_whatsapp_deployments USING btree (assistant_id);
CREATE UNIQUE INDEX idx_assistants_id ON public.assistants USING btree (id);
CREATE INDEX idx_knowledge_collections_knowledge_id ON public.knowledge_collections USING btree (knowledge_id);
CREATE INDEX idx_knowledge_collections_knowledge_id_status ON public.knowledge_collections USING btree (knowledge_id, status);
CREATE INDEX idx_knowledge_collections_status ON public.knowledge_collections USING btree (status);
CREATE INDEX idx_knowledge_documents_knowledge_id_project_id_organization_id ON public.knowledge_documents USING btree (knowledge_id, project_id, organization_id);
CREATE INDEX idx_knowledge_documents_organization_id ON public.knowledge_documents USING btree (organization_id);
CREATE INDEX idx_knowledge_documents_project_id ON public.knowledge_documents USING btree (project_id);
CREATE UNIQUE INDEX idx_knowledge_tags ON public.knowledge_tags USING btree (knowledge_id);
CREATE INDEX idx_knowledges_id ON public.knowledges USING btree (id);
CREATE INDEX idx_knowledges_id_status ON public.knowledges USING btree (id, status);
CREATE INDEX idx_knowledges_status ON public.knowledges USING btree (status);
CREATE INDEX idx_recordings_conversation_id ON public.assistant_conversation_recordings USING btree (assistant_conversation_id);