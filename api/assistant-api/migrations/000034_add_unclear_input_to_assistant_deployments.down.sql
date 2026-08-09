ALTER TABLE public.assistant_api_deployments DROP COLUMN IF EXISTS unclear_input_timeout;
ALTER TABLE public.assistant_api_deployments DROP COLUMN IF EXISTS unclear_input_message;

ALTER TABLE public.assistant_debugger_deployments DROP COLUMN IF EXISTS unclear_input_timeout;
ALTER TABLE public.assistant_debugger_deployments DROP COLUMN IF EXISTS unclear_input_message;

ALTER TABLE public.assistant_phone_deployments DROP COLUMN IF EXISTS unclear_input_timeout;
ALTER TABLE public.assistant_phone_deployments DROP COLUMN IF EXISTS unclear_input_message;

ALTER TABLE public.assistant_web_plugin_deployments DROP COLUMN IF EXISTS unclear_input_timeout;
ALTER TABLE public.assistant_web_plugin_deployments DROP COLUMN IF EXISTS unclear_input_message;

ALTER TABLE public.assistant_whatsapp_deployments DROP COLUMN IF EXISTS unclear_input_timeout;
ALTER TABLE public.assistant_whatsapp_deployments DROP COLUMN IF EXISTS unclear_input_message;
