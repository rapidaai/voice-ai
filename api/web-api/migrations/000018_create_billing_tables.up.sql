BEGIN;

CREATE TABLE public.billing_plans (
    id bigint PRIMARY KEY CHECK (id > 0 AND id <= 9223372036854775807),
    name character varying(100) NOT NULL,
    slug character varying(50) NOT NULL UNIQUE,
    description text DEFAULT '' NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL CHECK (sort_order >= 0),
    price_monthly bigint DEFAULT 0 NOT NULL CHECK (price_monthly >= 0),
    price_yearly bigint DEFAULT 0 NOT NULL CHECK (price_yearly >= 0),
    currency character varying(10) DEFAULT 'usd' NOT NULL,
    stripe_url text DEFAULT '' NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    CONSTRAINT billing_plans_name_check CHECK (name = btrim(name) AND name <> ''),
    CONSTRAINT billing_plans_slug_check CHECK (slug = btrim(slug) AND slug <> ''),
    CONSTRAINT billing_plans_description_check CHECK (description = btrim(description)),
    CONSTRAINT billing_plans_currency_check CHECK (currency = btrim(currency) AND currency <> ''),
    CONSTRAINT billing_plans_stripe_url_check CHECK (stripe_url = btrim(stripe_url))
);

CREATE UNIQUE INDEX billing_plans_one_default_idx
    ON public.billing_plans (is_default) WHERE is_default;
CREATE INDEX billing_plans_active_order_idx
    ON public.billing_plans (is_active, sort_order, id);

CREATE TABLE public.billing_plan_quotas (
    id bigint PRIMARY KEY CHECK (id > 0 AND id <= 9223372036854775807),
    billing_plan_id bigint NOT NULL CHECK (billing_plan_id > 0),
    resource_type character varying(100) NOT NULL,
    quota_limit bigint DEFAULT 0 NOT NULL CHECK (quota_limit >= -1),
    unit character varying(32) NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    CONSTRAINT billing_plan_quotas_plan_fk FOREIGN KEY (billing_plan_id)
        REFERENCES public.billing_plans (id) ON DELETE RESTRICT,
    CONSTRAINT billing_plan_quotas_resource_check CHECK (
        resource_type = btrim(resource_type) AND resource_type <> ''
    ),
    CONSTRAINT billing_plan_quotas_unit_check CHECK (unit = btrim(unit) AND unit <> ''),
    CONSTRAINT billing_plan_quotas_plan_resource_unique UNIQUE (billing_plan_id, resource_type)
);

CREATE INDEX billing_plan_quotas_plan_order_idx
    ON public.billing_plan_quotas (billing_plan_id, resource_type, id);

CREATE TABLE public.billing_subscriptions (
    id bigint PRIMARY KEY CHECK (id > 0 AND id <= 9223372036854775807),
    organization_id bigint NOT NULL UNIQUE CHECK (organization_id > 0),
    billing_plan_id bigint NOT NULL CHECK (billing_plan_id > 0),
    billing_interval character varying(20) DEFAULT 'monthly' NOT NULL,
    current_period_start timestamp without time zone DEFAULT now() NOT NULL,
    current_period_end timestamp without time zone,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,
    status character varying(50) DEFAULT 'ACTIVE' NOT NULL,
    created_actor_type character varying(32) DEFAULT 'unknown' NOT NULL,
    created_actor_id bigint,
    updated_actor_type character varying(32),
    updated_actor_id bigint,
    CONSTRAINT billing_subscriptions_organization_fk FOREIGN KEY (organization_id)
        REFERENCES public.organizations (id) ON DELETE RESTRICT,
    CONSTRAINT billing_subscriptions_plan_fk FOREIGN KEY (billing_plan_id)
        REFERENCES public.billing_plans (id) ON DELETE RESTRICT,
    CONSTRAINT billing_subscriptions_interval_check CHECK (billing_interval IN ('monthly', 'yearly')),
    CONSTRAINT billing_subscriptions_period_check CHECK (
        current_period_end IS NULL OR current_period_end >= current_period_start
    ),
    CONSTRAINT billing_subscriptions_metadata_check CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT billing_subscriptions_status_check CHECK (status IN ('ACTIVE', 'ARCHIEVE')),
    CONSTRAINT billing_subscriptions_created_actor_check CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL) OR
        (created_actor_type IN ('user', 'project', 'organization', 'service', 'system')
            AND created_actor_id IS NOT NULL AND created_actor_id > 0)
    ),
    CONSTRAINT billing_subscriptions_updated_actor_check CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL) OR
        (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system')
            AND updated_actor_id IS NOT NULL AND updated_actor_id > 0)
    )
);

CREATE INDEX billing_subscriptions_plan_idx
    ON public.billing_subscriptions (billing_plan_id, organization_id);
CREATE INDEX billing_subscriptions_status_idx
    ON public.billing_subscriptions (status, organization_id);

COMMIT;
