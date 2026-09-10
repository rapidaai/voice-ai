BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

LOCK TABLE public.assistant_deployment_audios,
    public.assistant_deployment_audio_options IN SHARE ROW EXCLUSIVE MODE;

-- Preserve the canonical row, or promote the highest-priority existing alias.
UPDATE public.assistant_deployment_audio_options option
SET key = 'microphone.eos.' || mapping.canonical_key
FROM (VALUES
    ('livekit_eos', 'fallback_timeout', 'quick_timeout'),
    ('livekit_eos', 'timeout', 'quick_timeout'),
    ('livekit_eos', 'silence_timeout', 'extended_timeout'),
    ('pipecat_smart_turn_eos', 'timeout', 'fallback_timeout'),
    ('pipecat_smart_turn_eos', 'silence_timeout', 'extended_timeout')
) AS mapping(provider, legacy_key, canonical_key)
JOIN public.assistant_deployment_audio_options provider
    ON provider.key = 'microphone.eos.provider' AND provider.value = mapping.provider
JOIN public.assistant_deployment_audios deployment
    ON deployment.id = provider.assistant_deployment_audio_id AND deployment.audio_type = 'input'
WHERE option.assistant_deployment_audio_id = deployment.id
    AND option.key = 'microphone.eos.' || mapping.legacy_key
    AND NOT EXISTS (
        SELECT 1 FROM public.assistant_deployment_audio_options preferred
        WHERE preferred.assistant_deployment_audio_id = deployment.id
            AND (preferred.key = 'microphone.eos.' || mapping.canonical_key
                OR (mapping.provider = 'livekit_eos' AND mapping.legacy_key = 'timeout'
                    AND preferred.key = 'microphone.eos.fallback_timeout'))
    );

DELETE FROM public.assistant_deployment_audio_options option
USING public.assistant_deployment_audios deployment, public.assistant_deployment_audio_options provider
WHERE deployment.id = option.assistant_deployment_audio_id AND deployment.audio_type = 'input'
    AND provider.assistant_deployment_audio_id = deployment.id AND provider.key = 'microphone.eos.provider'
    AND ((provider.value = 'livekit_eos' AND option.key IN (
            'microphone.eos.fallback_timeout', 'microphone.eos.timeout', 'microphone.eos.silence_timeout'))
        OR (provider.value = 'pipecat_smart_turn_eos' AND option.key IN (
            'microphone.eos.timeout', 'microphone.eos.silence_timeout', 'microphone.eos.quick_timeout',
            'microphone.eos.max_history_turns', 'microphone.eos.model')));

COMMIT;
