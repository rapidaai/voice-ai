BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM public.billing_subscriptions) OR
       EXISTS (SELECT 1 FROM public.billing_plan_quotas) OR
       EXISTS (SELECT 1 FROM public.billing_plans) THEN
        RAISE EXCEPTION 'billing tables contain data; rollback 000019 or preserve the data before dropping the schema';
    END IF;
END
$$;

DROP TABLE public.billing_subscriptions;
DROP TABLE public.billing_plan_quotas;
DROP TABLE public.billing_plans;

COMMIT;
