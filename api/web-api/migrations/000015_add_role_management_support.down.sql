BEGIN;

ALTER TABLE public.role_permissions
    ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE public.role_permissions_id_seq;

ALTER TABLE public.role_definitions
    ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE public.role_definitions_id_seq;

COMMIT;
