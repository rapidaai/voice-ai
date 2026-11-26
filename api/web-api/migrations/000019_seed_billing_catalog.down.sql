BEGIN;

DO $$
BEGIN
    IF EXISTS (
        WITH expected(id, name, slug, description, is_default, is_active, sort_order, price_monthly, price_yearly, currency, stripe_url) AS (
            VALUES
                (1000000000000001::bigint, 'Free', 'free', 'Get started with the basics', true, true, 1, 0::bigint, 0::bigint, 'usd', ''),
                (1000000000000002::bigint, 'Pro', 'pro', 'For growing teams that need more', false, true, 2, 9900::bigint, 99000::bigint, 'usd', ''),
                (1000000000000003::bigint, 'Enterprise', 'enterprise', 'Custom solutions for large organizations', false, true, 3, 0::bigint, 0::bigint, 'usd', '')
        )
        SELECT 1
        FROM expected
        LEFT JOIN public.billing_plans AS plan USING (id)
        WHERE plan.id IS NULL OR plan.name IS DISTINCT FROM expected.name
            OR plan.slug IS DISTINCT FROM expected.slug
            OR plan.description IS DISTINCT FROM expected.description
            OR plan.is_default IS DISTINCT FROM expected.is_default
            OR plan.is_active IS DISTINCT FROM expected.is_active
            OR plan.sort_order IS DISTINCT FROM expected.sort_order
            OR plan.price_monthly IS DISTINCT FROM expected.price_monthly
            OR plan.price_yearly IS DISTINCT FROM expected.price_yearly
            OR plan.currency IS DISTINCT FROM expected.currency
            OR plan.stripe_url IS DISTINCT FROM expected.stripe_url
            OR plan.updated_date IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'seeded billing plans changed; preserve or restore them before rollback';
    END IF;

    IF EXISTS (
        WITH expected(id, billing_plan_id, resource_type, quota_limit, unit) AS (
            VALUES
                (2000000000000001::bigint, 1000000000000001::bigint, 'users', 1::bigint, 'count'),
                (2000000000000002::bigint, 1000000000000001::bigint, 'assistants', 1::bigint, 'count'),
                (2000000000000003::bigint, 1000000000000001::bigint, 'endpoints', 1::bigint, 'count'),
                (2000000000000004::bigint, 1000000000000001::bigint, 'knowledge_bases', 1::bigint, 'count'),
                (2000000000000005::bigint, 1000000000000001::bigint, 'log_retention_days', 30::bigint, 'day'),
                (2000000000000006::bigint, 1000000000000002::bigint, 'users', 5::bigint, 'count'),
                (2000000000000007::bigint, 1000000000000002::bigint, 'assistants', 5::bigint, 'count'),
                (2000000000000008::bigint, 1000000000000002::bigint, 'endpoints', 5::bigint, 'count'),
                (2000000000000009::bigint, 1000000000000002::bigint, 'knowledge_bases', 1::bigint, 'count'),
                (2000000000000010::bigint, 1000000000000002::bigint, 'log_retention_days', 30::bigint, 'day'),
                (2000000000000011::bigint, 1000000000000003::bigint, 'users', -1::bigint, 'count'),
                (2000000000000012::bigint, 1000000000000003::bigint, 'assistants', -1::bigint, 'count'),
                (2000000000000013::bigint, 1000000000000003::bigint, 'endpoints', -1::bigint, 'count'),
                (2000000000000014::bigint, 1000000000000003::bigint, 'knowledge_bases', -1::bigint, 'count'),
                (2000000000000015::bigint, 1000000000000003::bigint, 'log_retention_days', -1::bigint, 'day')
        )
        SELECT 1
        FROM expected
        LEFT JOIN public.billing_plan_quotas AS quota USING (id)
        WHERE quota.id IS NULL OR quota.billing_plan_id IS DISTINCT FROM expected.billing_plan_id
            OR quota.resource_type IS DISTINCT FROM expected.resource_type
            OR quota.quota_limit IS DISTINCT FROM expected.quota_limit
            OR quota.unit IS DISTINCT FROM expected.unit
            OR quota.updated_date IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'seeded billing quotas changed; preserve or restore them before rollback';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.billing_subscriptions AS subscription
        WHERE subscription.id = subscription.organization_id
          AND (
              subscription.metadata ->> 'source' IS DISTINCT FROM 'migration-000019' OR
              subscription.billing_plan_id <> 1000000000000001 OR
              subscription.billing_interval <> 'monthly' OR
              subscription.current_period_end IS NOT NULL OR
              subscription.current_period_start <> subscription.created_date OR
              subscription.status IS DISTINCT FROM subscription.metadata ->> 'initialStatus' OR
              subscription.metadata <> jsonb_build_object(
                  'source', 'migration-000019',
                  'initialStatus', subscription.metadata ->> 'initialStatus'
              ) OR
              subscription.created_actor_type <> 'unknown' OR
              subscription.created_actor_id IS NOT NULL OR
              subscription.updated_date IS NOT NULL OR
              subscription.updated_actor_type IS NOT NULL OR
              subscription.updated_actor_id IS NOT NULL
          )
    ) THEN
        RAISE EXCEPTION 'migration-owned billing subscriptions changed; preserve or restore them before rollback';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.billing_subscriptions AS subscription
        WHERE subscription.billing_plan_id IN (1000000000000001, 1000000000000002, 1000000000000003)
          AND subscription.metadata ->> 'source' IS DISTINCT FROM 'migration-000019'
    ) THEN
        RAISE EXCEPTION 'later billing subscriptions reference the seeded catalog; preserve them before rollback';
    END IF;
END
$$;

DELETE FROM public.billing_subscriptions
WHERE id = organization_id
  AND metadata ->> 'source' = 'migration-000019';

DELETE FROM public.billing_plan_quotas
WHERE id BETWEEN 2000000000000001 AND 2000000000000015;

DELETE FROM public.billing_plans
WHERE id IN (1000000000000001, 1000000000000002, 1000000000000003);

COMMIT;
