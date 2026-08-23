SET lock_timeout = '5s';

DROP TABLE public.system_identities;
DROP TABLE public.service_identities;

RESET lock_timeout;
