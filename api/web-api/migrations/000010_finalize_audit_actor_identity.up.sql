ALTER TABLE public.notification_settings VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.notification_settings VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.organizations VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.organizations VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.project_credentials VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.project_credentials VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.projects VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.projects VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.user_auth_tokens VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.user_auth_tokens VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.user_auths VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.user_auths VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.user_feature_permissions VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.user_feature_permissions VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.user_organization_roles VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.user_organization_roles VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.user_project_roles VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.user_project_roles VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.user_roles VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.user_roles VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.vaults VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.vaults VALIDATE CONSTRAINT audit_updated_actor_pair;
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
BEFORE UPDATE ON public.notification_settings
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.organizations
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.project_credentials
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.projects
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.user_auth_tokens
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.user_auths
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.user_feature_permissions
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.user_organization_roles
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.user_project_roles
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.user_roles
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.vaults
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.organization_credentials
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.service_identities
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.system_identities
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();


DROP PROCEDURE public.backfill_web_audit_actor_identity();
