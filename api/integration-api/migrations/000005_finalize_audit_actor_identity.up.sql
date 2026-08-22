ALTER TABLE public.external_audits VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.external_audits VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.external_audit_metadata VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.external_audit_metadata VALIDATE CONSTRAINT audit_updated_actor_pair;
CREATE OR REPLACE FUNCTION public.reject_created_actor_change()
RETURNS trigger
LANGUAGE plpgsql
AS $function$
BEGIN
    IF NEW.created_actor_type IS DISTINCT FROM OLD.created_actor_type
       OR NEW.created_actor_id IS DISTINCT FROM OLD.created_actor_id THEN
        RAISE EXCEPTION 'creation actor is immutable';
    END IF;
    RETURN NEW;
END
$function$;

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.external_audits
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.external_audit_metadata
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();


DROP PROCEDURE public.backfill_integration_audit_actor_identity();
