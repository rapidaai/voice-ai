--
-- PostgreSQL database dump
--

-- Dumped from database version 16.4 (Postgres.app)
-- Dumped by pg_dump version 16.4 (Postgres.app)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: generate_id(); Type: FUNCTION; Schema: public; Owner: prashant.srivastav
--

CREATE FUNCTION public.generate_id() RETURNS bigint
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN (random() * 2147483648)::int8 * 4294967296
           + (random() * 4294967296)::int8;
END;
$$;


ALTER FUNCTION public.generate_id() OWNER TO "prashant.srivastav";

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: leads; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.leads (
    id bigint NOT NULL,
    email character varying(200) NOT NULL,
    company_name character varying(200) NOT NULL,
    expected_volume character varying(200) NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    update_date timestamp without time zone,
    updated_date timestamp without time zone
);


ALTER TABLE public.leads OWNER TO "prashant.srivastav";

--
-- Name: o_auth_external_connects; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.o_auth_external_connects (
    id bigint NOT NULL,
    created_date timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_date timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    identifier character varying(200) NOT NULL,
    tool_connect character varying(200) NOT NULL,
    tool_id bigint NOT NULL,
    linker character varying(200) NOT NULL,
    linker_id bigint NOT NULL,
    redirect_to character varying(200) NOT NULL
);


ALTER TABLE public.o_auth_external_connects OWNER TO "prashant.srivastav";

--
-- Name: organizations; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.organizations (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    name character varying(200) NOT NULL,
    description character varying(400) NOT NULL,
    size character varying(100) NOT NULL,
    industry character varying(200) NOT NULL,
    contact character varying(200) NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);


ALTER TABLE public.organizations OWNER TO "prashant.srivastav";

--
-- Name: project_credentials; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.project_credentials (
    id bigint NOT NULL,
    organization_id bigint NOT NULL,
    project_id bigint NOT NULL,
    name character varying(200) NOT NULL,
    key character varying(400) NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);


ALTER TABLE public.project_credentials OWNER TO "prashant.srivastav";

--
-- Name: projects; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.projects (
    id bigint NOT NULL,
    organization_id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    name character varying(200) NOT NULL,
    description character varying(400) NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);


ALTER TABLE public.projects OWNER TO "prashant.srivastav";

--
-- Name: user_auth_tokens; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.user_auth_tokens (
    id bigint NOT NULL,
    user_auth_id bigint NOT NULL,
    token_type character varying(200) NOT NULL,
    token character varying(400) NOT NULL,
    expire_at timestamp without time zone DEFAULT now(),
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);


ALTER TABLE public.user_auth_tokens OWNER TO "prashant.srivastav";

--
-- Name: user_auths; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.user_auths (
    id bigint NOT NULL,
    name character varying(200) NOT NULL,
    email character varying(200) NOT NULL,
    password character varying(400) NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL,
    source character varying(50) DEFAULT 'direct'::character varying NOT NULL
);


ALTER TABLE public.user_auths OWNER TO "prashant.srivastav";

--
-- Name: user_feature_permissions; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.user_feature_permissions (
    id bigint NOT NULL,
    user_auth_id bigint NOT NULL,
    feature character varying(50) NOT NULL,
    is_enabled boolean,
    created_by bigint NOT NULL,
    updated_by bigint,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL
);


ALTER TABLE public.user_feature_permissions OWNER TO "prashant.srivastav";

--
-- Name: user_organization_roles; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.user_organization_roles (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    user_auth_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    role character varying(200) NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);


ALTER TABLE public.user_organization_roles OWNER TO "prashant.srivastav";

--
-- Name: user_project_roles; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.user_project_roles (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    project_id bigint NOT NULL,
    user_auth_id bigint NOT NULL,
    role character varying(200) NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);


ALTER TABLE public.user_project_roles OWNER TO "prashant.srivastav";

--
-- Name: user_roles; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.user_roles (
    id bigint NOT NULL,
    user_auth_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    role character varying(200) NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);


ALTER TABLE public.user_roles OWNER TO "prashant.srivastav";

--
-- Name: user_socials; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.user_socials (
    id bigint NOT NULL,
    user_auth_id bigint NOT NULL,
    social character varying(200) NOT NULL,
    identifier character varying(200) NOT NULL,
    verified boolean DEFAULT false,
    token character varying(500) NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL
);


ALTER TABLE public.user_socials OWNER TO "prashant.srivastav";

--
-- Name: vaults; Type: TABLE; Schema: public; Owner: prashant.srivastav
--

CREATE TABLE public.vaults (
    id bigint NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    vault_type character varying(200) NOT NULL,
    vault_type_id bigint NOT NULL,
    vault_level character varying(200) NOT NULL,
    vault_level_id bigint NOT NULL,
    name character varying(200) NOT NULL,
    value json NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);


ALTER TABLE public.vaults OWNER TO "prashant.srivastav";

--
-- Name: leads leads_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.leads
    ADD CONSTRAINT leads_pkey PRIMARY KEY (id);


--
-- Name: o_auth_external_connects o_auth_external_connects_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.o_auth_external_connects
    ADD CONSTRAINT o_auth_external_connects_pkey PRIMARY KEY (id);


--
-- Name: organizations organizations_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);



--
-- Name: user_auth_tokens user_auth_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.user_auth_tokens
    ADD CONSTRAINT user_auth_tokens_pkey PRIMARY KEY (id);


--
-- Name: user_auths user_auths_email_key; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.user_auths
    ADD CONSTRAINT user_auths_email_key UNIQUE (email);


--
-- Name: user_auths user_auths_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.user_auths
    ADD CONSTRAINT user_auths_pkey PRIMARY KEY (id);


--
-- Name: user_organization_roles user_organization_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.user_organization_roles
    ADD CONSTRAINT user_organization_roles_pkey PRIMARY KEY (id);


--
-- Name: user_project_roles user_project_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.user_project_roles
    ADD CONSTRAINT user_project_roles_pkey PRIMARY KEY (id);


--
-- Name: user_roles user_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (id);


--
-- Name: user_socials user_socials_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.user_socials
    ADD CONSTRAINT user_socials_pkey PRIMARY KEY (id);


--
-- Name: vaults vaults_pkey; Type: CONSTRAINT; Schema: public; Owner: prashant.srivastav
--

ALTER TABLE ONLY public.vaults
    ADD CONSTRAINT vaults_pkey PRIMARY KEY (id);




--
-- Name: idx_projects_id; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX idx_projects_id ON public.projects USING btree (id);


--
-- Name: idx_projects_organization_id_status; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX idx_projects_organization_id_status ON public.projects USING btree (organization_id, status);


--
-- Name: idx_user_project_roles_project_id; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX idx_user_project_roles_project_id ON public.user_project_roles USING btree (project_id);


--
-- Name: idx_user_project_roles_user_auth_id_status; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX idx_user_project_roles_user_auth_id_status ON public.user_project_roles USING btree (user_auth_id, status);


--
-- Name: idx_vault_level; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX idx_vault_level ON public.vaults USING btree (vault_level);


--
-- Name: idx_vault_level_id; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX idx_vault_level_id ON public.vaults USING btree (vault_level_id);


--
-- Name: idx_vault_type; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX idx_vault_type ON public.vaults USING btree (vault_type);


--
-- Name: idx_vault_type_id; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX idx_vault_type_id ON public.vaults USING btree (vault_type_id);


--
-- Name: ua_idx_email; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX ua_idx_email ON public.user_auths USING btree (email);


--
-- Name: up_idx_auth_id; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX up_idx_auth_id ON public.user_auth_tokens USING btree (user_auth_id);


--
-- Name: ur_idx_auth_id; Type: INDEX; Schema: public; Owner: prashant.srivastav
--

CREATE INDEX ur_idx_auth_id ON public.user_roles USING btree (user_auth_id);


--
-- PostgreSQL database dump complete
--

