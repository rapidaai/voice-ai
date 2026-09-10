import { loadProviderConfig } from '../config-loader';

describe('EOS provider config', () => {
  it.each([
    [
      'pipecat_smart_turn_eos',
      {
        fallback_timeout: '500',
        threshold: '0.5',
        extended_timeout: '3000',
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
});
