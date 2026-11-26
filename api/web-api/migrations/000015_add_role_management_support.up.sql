BEGIN;

CREATE SEQUENCE public.role_definitions_id_seq AS bigint;
ALTER SEQUENCE public.role_definitions_id_seq OWNED BY public.role_definitions.id;
SELECT setval(
    'public.role_definitions_id_seq',
    GREATEST(COALESCE((SELECT max(id) FROM public.role_definitions), 0) + 1, 1),
    false
);
ALTER TABLE public.role_definitions
    ALTER COLUMN id SET DEFAULT nextval('public.role_definitions_id_seq');

CREATE SEQUENCE public.role_permissions_id_seq AS bigint;
ALTER SEQUENCE public.role_permissions_id_seq OWNED BY public.role_permissions.id;
SELECT setval(
    'public.role_permissions_id_seq',
    GREATEST(COALESCE((SELECT max(id) FROM public.role_permissions), 0) + 1, 1),
    false
);
ALTER TABLE public.role_permissions
    ALTER COLUMN id SET DEFAULT nextval('public.role_permissions_id_seq');

COMMIT;
