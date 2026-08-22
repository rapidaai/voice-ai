SET lock_timeout = '5s';

CREATE TABLE IF NOT EXISTS public.audit_actor_migration_metrics (
    table_name text PRIMARY KEY,
    processed_rows bigint NOT NULL DEFAULT 0,
    user_classified_rows bigint NOT NULL DEFAULT 0,
    unknown_classified_rows bigint NOT NULL DEFAULT 0,
    failed_rows bigint NOT NULL DEFAULT 0,
    remaining_rows bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE PROCEDURE public.backfill_endpoint_audit_actor_identity()
LANGUAGE plpgsql
AS $migration$
<<backfill>>
DECLARE
    audited_table text;
    changed_rows integer;
    total_rows bigint;
    user_classified_rows bigint;
    unknown_classified_rows bigint;
    failed_rows bigint;
    remaining_rows bigint;
BEGIN
    PERFORM set_config('lock_timeout', '5s', false);
    FOREACH audited_table IN ARRAY ARRAY[
        'endpoint_cachings',
        'endpoint_log_arguments',
        'endpoint_log_metadata',
        'endpoint_log_metrics',
        'endpoint_log_options',
        'endpoint_provider_model_options',
        'endpoint_provider_models',
        'endpoint_retries',
        'endpoint_tags',
        'endpoints'
    ]
    LOOP
        total_rows := 0;
        LOOP
            EXECUTE format($sql$
                WITH batch AS (
                    SELECT ctid
                    FROM public.%I
                    WHERE created_actor_type IS NULL
                       OR (updated_by IS NOT NULL AND updated_actor_type IS NULL)
                    LIMIT 10000
                    FOR UPDATE SKIP LOCKED
                )
                UPDATE public.%I AS target
                SET created_actor_type = CASE
                        WHEN target.created_actor_type IS NOT NULL THEN target.created_actor_type
                        WHEN target.created_by > 0 THEN 'user'
                        ELSE 'unknown'
                    END,
                    created_actor_id = CASE
                        WHEN target.created_actor_type IS NOT NULL THEN target.created_actor_id
                        WHEN target.created_by > 0 THEN target.created_by
                        ELSE NULL
                    END,
                    updated_actor_type = CASE
                        WHEN target.updated_actor_type IS NOT NULL THEN target.updated_actor_type
                        WHEN target.updated_by > 0 THEN 'user'
                        WHEN target.updated_by IS NULL THEN NULL
                        ELSE 'unknown'
                    END,
                    updated_actor_id = CASE
                        WHEN target.updated_actor_type IS NOT NULL THEN target.updated_actor_id
                        WHEN target.updated_by > 0 THEN target.updated_by
                        ELSE NULL
                    END
                FROM batch
                WHERE target.ctid = batch.ctid
            $sql$, audited_table, audited_table);
            GET DIAGNOSTICS changed_rows = ROW_COUNT;
            total_rows := total_rows + changed_rows;
            INSERT INTO public.audit_actor_migration_metrics (table_name, processed_rows, updated_at)
            VALUES (audited_table, changed_rows, CURRENT_TIMESTAMP)
            ON CONFLICT (table_name) DO UPDATE
            SET processed_rows = public.audit_actor_migration_metrics.processed_rows + EXCLUDED.processed_rows,
                updated_at = EXCLUDED.updated_at;
            RAISE LOG 'audit actor backfill table=% batch_rows=% cumulative_rows=%', audited_table, changed_rows, total_rows;
            COMMIT;
            EXIT WHEN changed_rows = 0;
        END LOOP;
        EXECUTE format(
            'SELECT count(*) FILTER (WHERE created_actor_type = ''user''), count(*) FILTER (WHERE created_actor_type = ''unknown''), count(*) FILTER (WHERE NOT ((created_actor_type = ''unknown'' AND created_actor_id IS NULL) OR (created_actor_type IN (''user'', ''project'', ''organization'', ''service'', ''system'') AND created_actor_id BETWEEN 1 AND 9223372036854775807)) OR NOT ((updated_actor_type IS NULL AND updated_actor_id IS NULL) OR (updated_actor_type = ''unknown'' AND updated_actor_id IS NULL) OR (updated_actor_type IN (''user'', ''project'', ''organization'', ''service'', ''system'') AND updated_actor_id BETWEEN 1 AND 9223372036854775807))), count(*) FILTER (WHERE created_actor_type IS NULL) FROM public.%I',
            audited_table
        ) INTO user_classified_rows, unknown_classified_rows, failed_rows, remaining_rows;
        UPDATE public.audit_actor_migration_metrics
        SET user_classified_rows = backfill.user_classified_rows,
            unknown_classified_rows = backfill.unknown_classified_rows,
            failed_rows = backfill.failed_rows,
            remaining_rows = backfill.remaining_rows,
            updated_at = CURRENT_TIMESTAMP
        WHERE table_name = audited_table;
        RAISE LOG 'audit actor backfill complete table=% processed=% user_classified=% unknown_classified=% failed=% remaining=%', audited_table, total_rows, user_classified_rows, unknown_classified_rows, failed_rows, remaining_rows;
    END LOOP;
END
$migration$;

ALTER TABLE public.endpoint_cachings
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.endpoint_log_arguments
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.endpoint_log_metadata
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.endpoint_log_metrics
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.endpoint_log_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.endpoint_provider_model_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.endpoint_provider_models
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.endpoint_retries
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.endpoint_tags
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.endpoints
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;
