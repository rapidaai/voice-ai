CREATE TABLE public.external_audit_metadata (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    external_audit_id bigint NOT NULL,
    key character varying(200) NOT NULL,
    value character varying(1000) NOT NULL
);



CREATE TABLE public.integration_external_audits (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    integration_name character varying(200) NOT NULL,
    asset_prefix character varying(200) NOT NULL,
    response_status integer NOT NULL,
    time_taken bigint,
    credential_id bigint NOT NULL,
    project_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL
);


ALTER TABLE public.integration_external_audits OWNER TO "prashant.srivastav";

--
-- Name: external_audit_metadata external_audit_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.external_audit_metadata
    ADD CONSTRAINT external_audit_metadata_pkey PRIMARY KEY (id);


--
-- Name: external_audit_metadata external_audit_metadata_unique_constraint; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.external_audit_metadata
    ADD CONSTRAINT external_audit_metadata_unique_constraint UNIQUE (external_audit_id, key);


--
-- Name: external_audit_metadatas external_audit_metadatas_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.external_audit_metadatas
    ADD CONSTRAINT external_audit_metadatas_pkey PRIMARY KEY (id);


--
-- Name: external_audits external_audits_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.external_audits
    ADD CONSTRAINT external_audits_pkey PRIMARY KEY (id);


--
-- Name: integration_external_audit_metadatas integration_external_audit_metadatas_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.integration_external_audit_metadatas
    ADD CONSTRAINT integration_external_audit_metadatas_pkey PRIMARY KEY (id);


--
-- Name: integration_external_audits integration_external_audits_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.integration_external_audits
    ADD CONSTRAINT integration_external_audits_pkey PRIMARY KEY (id);


--
-- PostgreSQL database dump complete
--

