CREATE TABLE organizations (
    id bigint NOT NULL PRIMARY KEY,
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

CREATE TABLE user_organization_roles (
     id bigint NOT NULL PRIMARY KEY,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    user_auth_id bigint NOT NULL,
    organization_id bigint NOT NULL,
    role character varying(200) NOT NULL, 
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);

CREATE TABLE projects (
   id bigint NOT NULL PRIMARY KEY,
   organization_id bigint NOT NULL,
   created_date timestamp without time zone DEFAULT now() NOT NULL,
   updated_date timestamp without time zone,
   name character varying(200) NOT NULL, 
   description character varying(400) NOT NULL,
   status character varying(50) DEFAULT 'active'::character varying NOT NULL,
   created_by bigint NOT NULL,
   updated_by bigint NOT NULL
);

CREATE TABLE user_project_roles (
    id bigint NOT NULL PRIMARY KEY,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    project_id bigint NOT NULL,
    user_auth_id  bigint NOT NULL,
    role character varying(200) NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint NOT NULL,
    updated_by bigint NOT NULL
);

CREATE TABLE user_auths (
  id bigint NOT NULL PRIMARY KEY,
  name character varying(200) NOT NULL,
  email character varying(200) NOT NULL UNIQUE, 
  password character varying(400) NOT NULL,
  status character varying(50) DEFAULT 'active'::character varying NOT NULL,
  created_date timestamp without time zone DEFAULT now() NOT NULL,
  updated_date timestamp without time zone,
  created_by bigint NOT NULL,
  updated_by bigint NOT NULL,
  source character varying(50) DEFAULT 'direct'::character varying NOT NULL
);

CREATE TABLE user_auth_tokens (
  id bigint NOT NULL PRIMARY KEY,
  user_auth_id bigint NOT NULL,
  token_type character varying(200) NOT NULL,
  token character varying(400) NOT NULL,
  expire_at timestamp  without time zone DEFAULT now(),
  created_date timestamp without time zone DEFAULT now() NOT NULL,
  updated_date timestamp without time zone,
  status character varying(50) DEFAULT 'active'::character varying NOT NULL,
  created_by bigint NOT NULL,
  updated_by bigint NOT NULL
);


CREATE TABLE user_socials (
  id bigint NOT NULL PRIMARY KEY,
  user_auth_id bigint NOT NULL,
  social character varying(200) NOT NULL,
  identifier character varying(200) NOT NULL,
  verified boolean DEFAULT False,
  token character varying(500) NOT NULL,
  created_date timestamp without time zone DEFAULT now() NOT NULL,
  updated_date timestamp without time zone,
  status character varying(50) DEFAULT 'active'::character varying NOT NULL
);


CREATE TABLE user_roles (
  id bigint NOT NULL PRIMARY KEY,
  user_auth_id bigint NOT NULL,
  organization_id bigint NOT NULL,
  role character varying(200) NOT NULL,
  created_date timestamp without time zone DEFAULT now() NOT NULL,
  updated_date timestamp without time zone,
  status character varying(50) DEFAULT 'active'::character varying NOT NULL,
  created_by bigint NOT NULL,
  updated_by bigint NOT NULL
);

CREATE TABLE vaults (
  id bigint NOT NULL PRIMARY KEY,
  organization_id bigint NOT NULL,
  provider_id bigint NOT NULL,
  name character varying(200) NOT NULL,
  key  character varying(200) NOT NULL,
  created_date timestamp without time zone DEFAULT now() NOT NULL,
  updated_date timestamp without time zone,
  status character varying(50) DEFAULT 'active'::character varying NOT NULL,
  created_by bigint NOT NULL,
  updated_by bigint NOT NULL
);


CREATE TABLE leads (
  id bigint NOT NULL PRIMARY KEY,
  email character varying(200) NOT NULL,
  status character varying(50) DEFAULT 'active'::character varying NOT NULL,
  created_date timestamp without time zone DEFAULT now() NOT NULL,
  updated_date timestamp without time zone
);


CREATE INDEX IF NOT EXISTS ua_idx_email ON user_auths(email);

CREATE INDEX IF NOT EXISTS up_idx_auth_id ON user_auth_tokens(user_auth_id);

CREATE INDEX IF NOT EXISTS ur_idx_auth_id ON user_roles(user_auth_id);
