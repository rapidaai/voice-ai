SET lock_timeout = '5s';

ALTER TABLE public.endpoint_cachings
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.endpoint_log_arguments
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.endpoint_log_metadata
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.endpoint_log_metrics
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.endpoint_log_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.endpoint_provider_model_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.endpoint_provider_models
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.endpoint_retries
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.endpoint_tags
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.endpoints
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;
