ALTER TABLE public.endpoint_cachings VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoint_cachings VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.endpoint_log_arguments VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoint_log_arguments VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.endpoint_log_metadata VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoint_log_metadata VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.endpoint_log_metrics VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoint_log_metrics VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.endpoint_log_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoint_log_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.endpoint_provider_model_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoint_provider_model_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.endpoint_provider_models VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoint_provider_models VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.endpoint_retries VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoint_retries VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.endpoint_tags VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoint_tags VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.endpoints VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.endpoints VALIDATE CONSTRAINT audit_updated_actor_pair;
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
BEFORE UPDATE ON public.endpoint_cachings
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.endpoint_log_arguments
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.endpoint_log_metadata
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.endpoint_log_metrics
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.endpoint_log_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.endpoint_provider_model_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.endpoint_provider_models
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.endpoint_retries
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.endpoint_tags
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.endpoints
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();
