SET lock_timeout = '5s';

DROP TABLE IF EXISTS public.system_identities;
DROP TABLE IF EXISTS public.service_identities;

RESET lock_timeout;
