SET lock_timeout = '5s';

ALTER TABLE public.external_audits
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;


ALTER TABLE public.external_audit_metadata
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id IS NOT NULL)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id IS NOT NULL)
    ) NOT VALID;
