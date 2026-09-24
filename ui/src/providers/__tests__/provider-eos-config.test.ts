import { loadProviderConfig } from '../config-loader';

describe('EOS provider config', () => {
  it.each([
    [
      'pipecat_smart_turn_eos',
      {
        fallback_timeout: '1000',
        threshold: '0.85',
        extended_timeout: '4000',
      },
    ],
    [
      'livekit_eos',
      {
        threshold: '0.0289',
        quick_timeout: '250',
        extended_timeout: '3000',
        max_history_turns: '6',
        model: 'en',
      },
    ],
  ] as [string, Record<string, string>][])(
    '%s exposes only consumed options with backend defaults',
    (provider, expected) => {
      const parameters = loadProviderConfig(provider)?.eos?.parameters;
      expect(parameters).toBeDefined();
      expect(
        Object.fromEntries(
          parameters!.map(parameter => [
            parameter.key,
            String(parameter.default),
          ]),
        ),
      ).toEqual(
        Object.fromEntries(
          Object.entries(expected).map(([key, value]) => [
            `microphone.eos.${key}`,
            value,
          ]),
        ),
      );
      for (const parameter of parameters!) {
        if (parameter.key.endsWith('_timeout')) {
          expect(parameter.label).toContain('(ms)');
        }
      }
    },
  );

  it('LiveKit history length uses an integer number input without an arbitrary maximum', () => {
    const parameter = loadProviderConfig('livekit_eos')?.eos?.parameters.find(
      parameter => parameter.key === 'microphone.eos.max_history_turns',
    );
    expect(parameter).toMatchObject({ type: 'number', min: 1, step: 1 });
    expect(parameter).not.toHaveProperty('max');
  });

  it('does not expose legacy aliases or backend model path overrides as controls', () => {
    const blockedKeys = new Set([
      'microphone.eos.timeout',
      'microphone.eos.silence_timeout',
      'microphone.eos.quick_timeout',
      'microphone.eos.max_history_turns',
      'microphone.eos.model',
      'microphone.eos.livekit.model_path',
      'microphone.eos.livekit.tokenizer_path',
      'microphone.eos.pipecat.model_path',
    ]);

    const pipecatKeys =
      loadProviderConfig('pipecat_smart_turn_eos')?.eos?.parameters.map(
        parameter => parameter.key,
      ) ?? [];

    expect(pipecatKeys.filter(key => blockedKeys.has(key))).toEqual([]);
    expect(
      loadProviderConfig('livekit_eos')?.eos?.parameters.some(parameter =>
        parameter.key.endsWith('model_path'),
      ),
    ).toBe(false);
  });
});
