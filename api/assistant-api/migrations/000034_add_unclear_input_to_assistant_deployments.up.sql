ALTER TABLE public.assistant_api_deployments ADD COLUMN unclear_input_timeout DOUBLE PRECISION;
ALTER TABLE public.assistant_api_deployments ADD COLUMN unclear_input_message CHARACTER VARYING(500);

ALTER TABLE public.assistant_debugger_deployments ADD COLUMN unclear_input_timeout DOUBLE PRECISION;
ALTER TABLE public.assistant_debugger_deployments ADD COLUMN unclear_input_message CHARACTER VARYING(500);

ALTER TABLE public.assistant_phone_deployments ADD COLUMN unclear_input_timeout DOUBLE PRECISION;
ALTER TABLE public.assistant_phone_deployments ADD COLUMN unclear_input_message CHARACTER VARYING(500);

ALTER TABLE public.assistant_web_plugin_deployments ADD COLUMN unclear_input_timeout DOUBLE PRECISION;
ALTER TABLE public.assistant_web_plugin_deployments ADD COLUMN unclear_input_message CHARACTER VARYING(500);

ALTER TABLE public.assistant_whatsapp_deployments ADD COLUMN unclear_input_timeout DOUBLE PRECISION;
ALTER TABLE public.assistant_whatsapp_deployments ADD COLUMN unclear_input_message CHARACTER VARYING(500);
