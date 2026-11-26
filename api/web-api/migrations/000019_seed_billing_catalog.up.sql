BEGIN;

INSERT INTO public.billing_plans (
    id, name, slug, description, is_default, is_active, sort_order,
    price_monthly, price_yearly, currency, stripe_url
)
VALUES
    (1000000000000001, 'Free', 'free', 'Get started with the basics', true, true, 1, 0, 0, 'usd', ''),
    (1000000000000002, 'Pro', 'pro', 'For growing teams that need more', false, true, 2, 9900, 99000, 'usd', ''),
    (1000000000000003, 'Enterprise', 'enterprise', 'Custom solutions for large organizations', false, true, 3, 0, 0, 'usd', '');

INSERT INTO public.billing_plan_quotas (
    id, billing_plan_id, resource_type, quota_limit, unit
)
VALUES
    (2000000000000001, 1000000000000001, 'users', 1, 'count'),
    (2000000000000002, 1000000000000001, 'assistants', 1, 'count'),
    (2000000000000003, 1000000000000001, 'endpoints', 1, 'count'),
    (2000000000000004, 1000000000000001, 'knowledge_bases', 1, 'count'),
    (2000000000000005, 1000000000000001, 'log_retention_days', 30, 'day'),
    (2000000000000006, 1000000000000002, 'users', 5, 'count'),
    (2000000000000007, 1000000000000002, 'assistants', 5, 'count'),
    (2000000000000008, 1000000000000002, 'endpoints', 5, 'count'),
    (2000000000000009, 1000000000000002, 'knowledge_bases', 1, 'count'),
    (2000000000000010, 1000000000000002, 'log_retention_days', 30, 'day'),
    (2000000000000011, 1000000000000003, 'users', -1, 'count'),
    (2000000000000012, 1000000000000003, 'assistants', -1, 'count'),
    (2000000000000013, 1000000000000003, 'endpoints', -1, 'count'),
    (2000000000000014, 1000000000000003, 'knowledge_bases', -1, 'count'),
    (2000000000000015, 1000000000000003, 'log_retention_days', -1, 'day');

INSERT INTO public.billing_subscriptions (
    id, organization_id, billing_plan_id, billing_interval,
    current_period_start, metadata, created_date, status,
    created_actor_type, created_actor_id
)
SELECT
    organization.id,
    organization.id,
    1000000000000001,
    'monthly',
    statement_timestamp(),
    jsonb_build_object(
        'source', 'migration-000019',
        'initialStatus', CASE WHEN organization.status = 'ARCHIEVE' THEN 'ARCHIEVE' ELSE 'ACTIVE' END
    ),
    statement_timestamp(),
    CASE WHEN organization.status = 'ARCHIEVE' THEN 'ARCHIEVE' ELSE 'ACTIVE' END,
    'unknown',
    NULL
FROM public.organizations AS organization;

COMMIT;
