import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';
import { EndOfSpeech, VAD } from '@/providers';
import { loadProviderConfig } from '@/providers/config-loader';

jest.mock('@/app/components/providers', () => ({}));
jest.mock('@/app/components/providers/config-renderer', () => ({
  ConfigRenderer: () => null,
}));

const {
  GetDefaultVADConfig,
} = require('@/app/components/providers/vad/provider');
const {
  GetDefaultEOSConfig,
} = require('@/app/components/providers/end-of-speech/provider');
const {
  GetDefaultMicrophoneConfig,
} = require('@/app/components/providers/speech-to-text/provider');
const {
  GetDefaultNoiseCancellationConfig,
} = require('@/app/components/providers/noise-removal/provider');

const backendVadDefaults: Record<string, Record<string, string>> = {
  silero_vad: {
    'microphone.vad.confidence': '0.7',
    'microphone.vad.start_secs': '0.2',
    'microphone.vad.stop_secs': '0.2',
    'microphone.vad.min_volume': '0.6',
  },
  ten_vad: {
    'microphone.vad.confidence': '0.7',
    'microphone.vad.start_secs': '0.2',
    'microphone.vad.stop_secs': '0.2',
  },
  firered_vad: {
    'microphone.vad.confidence': '0.7',
    'microphone.vad.start_secs': '0.2',
    'microphone.vad.stop_secs': '0.2',
  },
};

const backendEosDefaults: Record<string, Record<string, string>> = {
  silence_based_eos: {
    'microphone.eos.timeout': '700',
  },
  livekit_eos: {
    'microphone.eos.threshold': '0.0289',
    'microphone.eos.quick_timeout': '250',
    'microphone.eos.extended_timeout': '3000',
    'microphone.eos.model': 'en',
    'microphone.eos.max_history_turns': '6',
  },
  pipecat_smart_turn_eos: {
    'microphone.eos.fallback_timeout': '500',
    'microphone.eos.threshold': '0.5',
    'microphone.eos.extended_timeout': '3000',
  },
};

const createMetadata = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

const cloneMetadata = (source: Metadata[]): Metadata[] =>
  source.map(m => createMetadata(m.getKey(), m.getValue()));

const normalizeMetadata = (source: Metadata[]): string[] =>
  source
    .map(m => `${m.getKey()}=${m.getValue()}`)
    .sort((a, b) => a.localeCompare(b));

const getMetadataValue = (source: Metadata[], key: string): string => {
  const value = source.find(m => m.getKey() === key)?.getValue();
  return value === undefined || value === null ? '' : String(value);
};

const expectedGetDefaultVADConfig = (
  provider: string,
  current: Metadata[],
): Metadata[] => {
  const defaults = backendVadDefaults[provider] || {};
  const nonVad = current.filter(m => !m.getKey().startsWith('microphone.vad.'));

  const vadParams: Metadata[] = [];
  const providerMeta = new Metadata();
  providerMeta.setKey('microphone.vad.provider');
  providerMeta.setValue(provider);
  vadParams.push(providerMeta);

  for (const [key, defaultValue] of Object.entries(defaults)) {
    const metadata = SetMetadata(current, key, defaultValue);
    if (metadata) vadParams.push(metadata);
  }

  return [...nonVad, ...vadParams];
};

const expectedGetDefaultEosConfig = (
  provider: string,
  current: Metadata[],
): Metadata[] => {
  const defaults = backendEosDefaults[provider] || {};
  const nonEos = current.filter(m => !m.getKey().startsWith('microphone.eos.'));

  const eosParams: Metadata[] = [];
  const providerMeta = new Metadata();
  providerMeta.setKey('microphone.eos.provider');
  providerMeta.setValue(provider);
  eosParams.push(providerMeta);

  for (const [key, defaultValue] of Object.entries(defaults)) {
    const metadata = SetMetadata(current, key, defaultValue);
    if (metadata) eosParams.push(metadata);
  }

  return [...nonEos, ...eosParams];
};

describe('Audio input advanced defaults parity', () => {
  it('all active VAD providers are config-driven', () => {
    expect(VAD().length).toBeGreaterThan(0);
    for (const provider of VAD()) {
      expect(loadProviderConfig(provider.code)?.vad).toBeDefined();
    }
  });

  it('all active VAD parameters include toggletip help text', () => {
    expect(VAD().length).toBeGreaterThan(0);
    for (const provider of VAD()) {
      const params = loadProviderConfig(provider.code)?.vad?.parameters ?? [];
      expect(params.length).toBeGreaterThan(0);
      for (const param of params) {
        expect(param.helpText).toEqual(expect.any(String));
        expect(param.helpText?.trim()).not.toHaveLength(0);
        expect(param.helpTextDisplay).toBe('toggletip');
      }
    }
  });

  it('all active end-of-speech providers are config-driven', () => {
    expect(EndOfSpeech().length).toBeGreaterThan(0);
    for (const provider of EndOfSpeech()) {
      expect(loadProviderConfig(provider.code)?.eos).toBeDefined();
    }
  });

  it.each(['silero_vad', 'ten_vad', 'firered_vad'])(
    '%s VAD defaults stay in parity with backend defaults',
    provider => {
      const seed = [
        createMetadata('rapida.credential_id', 'cred'),
        createMetadata('listen.model', 'nova-3'),
        createMetadata('microphone.vad.provider', 'ten_vad'),
        createMetadata('microphone.vad.confidence', '0.77'),
        createMetadata('microphone.vad.start_secs', '0.04'),
        createMetadata('microphone.vad.stop_secs', '0.15'),
      ];

      const expected = expectedGetDefaultVADConfig(
        provider,
        cloneMetadata(seed),
      );
      const current = GetDefaultVADConfig(provider, cloneMetadata(seed));
      expect(normalizeMetadata(current)).toEqual(normalizeMetadata(expected));
    },
  );

  it.each(['silence_based_eos', 'livekit_eos', 'pipecat_smart_turn_eos'])(
    '%s EOS preserves saved canonical values with backend-aligned options',
    provider => {
      const seed = [
        createMetadata('rapida.credential_id', 'cred'),
        createMetadata('listen.model', 'nova-3'),
        createMetadata('microphone.eos.provider', 'silence_based_eos'),
        createMetadata('microphone.eos.fallback_timeout', '950'),
        createMetadata('microphone.eos.threshold', '0.02'),
        createMetadata('microphone.eos.quick_timeout', '120'),
        createMetadata('microphone.eos.extended_timeout', '1800'),
        createMetadata('microphone.eos.model', 'custom-model'),
      ];

      const expected = expectedGetDefaultEosConfig(
        provider,
        cloneMetadata(seed),
      );
      const current = GetDefaultEOSConfig(provider, cloneMetadata(seed));
      expect(normalizeMetadata(current)).toEqual(normalizeMetadata(expected));
    },
  );

  it('VAD provider key is always updated to the selected provider', () => {
    const seed = [
      createMetadata('microphone.vad.provider', 'ten_vad'),
      createMetadata('microphone.vad.confidence', '0.3'),
    ];
    const updated = GetDefaultVADConfig('silero_vad', cloneMetadata(seed));

    expect(getMetadataValue(updated, 'microphone.vad.provider')).toBe(
      'silero_vad',
    );
  });

  it('EOS provider key is always updated to the selected provider', () => {
    const seed = [
      createMetadata('microphone.eos.provider', 'livekit_eos'),
      createMetadata('microphone.eos.fallback_timeout', '1200'),
    ];
    const updated = GetDefaultEOSConfig(
      'pipecat_smart_turn_eos',
      cloneMetadata(seed),
    );

    expect(getMetadataValue(updated, 'microphone.eos.provider')).toBe(
      'pipecat_smart_turn_eos',
    );
  });

  it('pipecat EOS applies provider defaults when switching from non-eos-only state', () => {
    const seed = [
      createMetadata('listen.model', 'nova-3'),
      createMetadata('microphone.vad.provider', 'silero_vad'),
      createMetadata('microphone.eos.provider', 'silence_based_eos'),
      createMetadata('microphone.eos.fallback_timeout', '700'),
    ];

    const switched = GetDefaultEOSConfig(
      'pipecat_smart_turn_eos',
      cloneMetadata(seed).filter(
        m => !m.getKey().startsWith('microphone.eos.'),
      ),
    );

    expect(getMetadataValue(switched, 'microphone.eos.provider')).toBe(
      'pipecat_smart_turn_eos',
    );
    expect(getMetadataValue(switched, 'microphone.eos.fallback_timeout')).toBe(
      '500',
    );
    expect(getMetadataValue(switched, 'microphone.eos.threshold')).toBe('0.5');
    expect(getMetadataValue(switched, 'microphone.eos.quick_timeout')).toBe('');
    expect(getMetadataValue(switched, 'microphone.eos.extended_timeout')).toBe(
      '3000',
    );
  });

  it('microphone defaults use pipecat eos provider', () => {
    const defaults = GetDefaultMicrophoneConfig([]);
    expect(getMetadataValue(defaults, 'microphone.barge_in_trigger')).toBe(
      'vad',
    );
    expect(getMetadataValue(defaults, 'microphone.vad.confidence')).toBe('0.7');
    expect(getMetadataValue(defaults, 'microphone.vad.start_secs')).toBe('0.2');
    expect(getMetadataValue(defaults, 'microphone.vad.stop_secs')).toBe('0.2');
    expect(getMetadataValue(defaults, 'microphone.vad.min_volume')).toBe('0.6');
    expect(getMetadataValue(defaults, 'microphone.eos.provider')).toBe(
      'pipecat_smart_turn_eos',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.fallback_timeout')).toBe(
      '500',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.threshold')).toBe('0.5');
    expect(getMetadataValue(defaults, 'microphone.eos.quick_timeout')).toBe('');
    expect(getMetadataValue(defaults, 'microphone.eos.extended_timeout')).toBe(
      '3000',
    );
  });

  it('microphone defaults hydrate missing values for an existing eos provider', () => {
    const defaults = GetDefaultMicrophoneConfig([
      createMetadata('microphone.eos.provider', 'livekit_eos'),
    ]);

    expect(getMetadataValue(defaults, 'microphone.eos.provider')).toBe(
      'livekit_eos',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.fallback_timeout')).toBe(
      '',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.threshold')).toBe(
      '0.0289',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.quick_timeout')).toBe(
      '250',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.extended_timeout')).toBe(
      '3000',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.model')).toBe('en');
    expect(getMetadataValue(defaults, 'microphone.eos.max_history_turns')).toBe(
      '6',
    );
  });

  it.each(['livekit_eos', 'pipecat_smart_turn_eos'])(
    '%s hydrates backend defaults through the config loader',
    provider => {
      const defaults = GetDefaultEOSConfig(provider, []);
      for (const [key, value] of Object.entries(backendEosDefaults[provider])) {
        expect(getMetadataValue(defaults, key)).toBe(value);
        const parameter = loadProviderConfig(provider)?.eos?.parameters.find(
          parameter => parameter.key === key,
        );
        expect(String(parameter?.default)).toBe(value);
      }
    },
  );

  it.each([
    [
      'livekit_eos',
      [
        createMetadata('microphone.eos.quick_timeout', '0'),
        createMetadata('microphone.eos.extended_timeout', '0'),
        createMetadata('microphone.eos.threshold', '0'),
        createMetadata('microphone.eos.model', 'custom-model'),
        createMetadata('microphone.eos.max_history_turns', '1000'),
      ],
    ],
    [
      'pipecat_smart_turn_eos',
      [
        createMetadata('microphone.eos.fallback_timeout', '0'),
        createMetadata('microphone.eos.extended_timeout', '0'),
        createMetadata('microphone.eos.threshold', '0'),
      ],
    ],
  ] as [string, Metadata[]][])(
    '%s preserves canonical zero and manual values without mutating input',
    (provider, providerOptions) => {
      const seed = [
        createMetadata('listen.model', 'nova-3'),
        createMetadata('microphone.vad.provider', 'silero_vad'),
        createMetadata('rapida.credential_id', 'cred'),
        createMetadata('microphone.eos.provider', provider),
        createMetadata('microphone.eos.obsolete', 'unused'),
        ...providerOptions,
      ];
      const original = seed.map(metadata => metadata.toObject());

      const defaults = GetDefaultEOSConfig(provider, seed);

      for (const option of providerOptions) {
        expect(getMetadataValue(defaults, option.getKey())).toBe(
          option.getValue(),
        );
      }
      expect(seed.map(metadata => metadata.toObject())).toEqual(original);
      expect(defaults).toEqual(expect.arrayContaining(seed.slice(0, 3)));
      expect(getMetadataValue(defaults, 'microphone.eos.obsolete')).toBe('');
      expect(
        normalizeMetadata(GetDefaultEOSConfig(provider, defaults)),
      ).toEqual(normalizeMetadata(defaults));
    },
  );

  it.each([
    [
      'livekit_eos',
      [
        createMetadata('microphone.eos.fallback_timeout', '950'),
        createMetadata('microphone.eos.timeout', '700'),
        createMetadata('microphone.eos.silence_timeout', '2200'),
      ],
      {
        'microphone.eos.quick_timeout': '250',
        'microphone.eos.extended_timeout': '3000',
      },
    ],
    [
      'pipecat_smart_turn_eos',
      [
        createMetadata('microphone.eos.timeout', '700'),
        createMetadata('microphone.eos.silence_timeout', '2200'),
        createMetadata('microphone.eos.quick_timeout', '800'),
        createMetadata('microphone.eos.model', 'old-livekit-model'),
        createMetadata('microphone.eos.max_history_turns', '20'),
      ],
      {
        'microphone.eos.fallback_timeout': '500',
        'microphone.eos.extended_timeout': '3000',
      },
    ],
  ] as [string, Metadata[], Record<string, string>][])(
    '%s ignores migrated legacy aliases and uses provider defaults',
    (provider, legacyOptions, expectedDefaults) => {
      const defaults = GetDefaultEOSConfig(provider, [
        createMetadata('listen.model', 'nova-3'),
        createMetadata('microphone.eos.provider', provider),
        createMetadata('microphone.eos.obsolete', 'unused'),
        ...legacyOptions,
      ]);

      for (const [key, value] of Object.entries(expectedDefaults)) {
        expect(getMetadataValue(defaults, key)).toBe(value);
      }
      for (const option of legacyOptions) {
        expect(getMetadataValue(defaults, option.getKey())).toBe('');
      }
      expect(getMetadataValue(defaults, 'microphone.eos.obsolete')).toBe('');
      expect(getMetadataValue(defaults, 'listen.model')).toBe('nova-3');
    },
  );

  it('preserves backend EOS model path overrides through hydration', () => {
    const defaults = GetDefaultEOSConfig('livekit_eos', [
      createMetadata(
        'microphone.eos.livekit.model_path',
        '/models/livekit.onnx',
      ),
      createMetadata(
        'microphone.eos.livekit.tokenizer_path',
        '/models/tokenizer.json',
      ),
      createMetadata('microphone.eos.pipecat.model_path', '/models/pipecat'),
      createMetadata('microphone.eos.quick_timeout', '0'),
      createMetadata('microphone.eos.timeout', '700'),
    ]);

    expect(getMetadataValue(defaults, 'microphone.eos.quick_timeout')).toBe(
      '0',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.timeout')).toBe('');
    expect(
      getMetadataValue(defaults, 'microphone.eos.livekit.model_path'),
    ).toBe('/models/livekit.onnx');
    expect(
      getMetadataValue(defaults, 'microphone.eos.livekit.tokenizer_path'),
    ).toBe('/models/tokenizer.json');
    expect(
      getMetadataValue(defaults, 'microphone.eos.pipecat.model_path'),
    ).toBe('/models/pipecat');
  });

  it('microphone defaults preserve canonical zero and backend EOS model paths', () => {
    const defaults = GetDefaultMicrophoneConfig([
      createMetadata('microphone.eos.provider', 'livekit_eos'),
      createMetadata('microphone.eos.quick_timeout', '0'),
      createMetadata(
        'microphone.eos.livekit.model_path',
        '/models/livekit.onnx',
      ),
      createMetadata(
        'microphone.eos.livekit.tokenizer_path',
        '/models/tokenizer.json',
      ),
      createMetadata('microphone.eos.pipecat.model_path', '/models/pipecat'),
    ]);

    expect(getMetadataValue(defaults, 'microphone.eos.provider')).toBe(
      'livekit_eos',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.quick_timeout')).toBe(
      '0',
    );
    expect(
      getMetadataValue(defaults, 'microphone.eos.livekit.model_path'),
    ).toBe('/models/livekit.onnx');
    expect(
      getMetadataValue(defaults, 'microphone.eos.livekit.tokenizer_path'),
    ).toBe('/models/tokenizer.json');
    expect(
      getMetadataValue(defaults, 'microphone.eos.pipecat.model_path'),
    ).toBe('/models/pipecat');
  });

  it('Pipecat removes quick timeout without assigning it to another budget', () => {
    const defaults = GetDefaultEOSConfig('pipecat_smart_turn_eos', [
      createMetadata('microphone.eos.quick_timeout', '800'),
      createMetadata('microphone.eos.extended_timeout', '2000'),
    ]);

    expect(getMetadataValue(defaults, 'microphone.eos.quick_timeout')).toBe('');
    expect(getMetadataValue(defaults, 'microphone.eos.fallback_timeout')).toBe(
      '500',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.extended_timeout')).toBe(
      '2000',
    );
  });

  it('LiveKit preserves saved model and history values without imposing a history cap', () => {
    const defaults = GetDefaultEOSConfig('livekit_eos', [
      createMetadata('microphone.eos.quick_timeout', '350'),
      createMetadata('microphone.eos.model', 'multilingual'),
      createMetadata('microphone.eos.max_history_turns', '1000'),
    ]);

    expect(getMetadataValue(defaults, 'microphone.eos.quick_timeout')).toBe(
      '350',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.model')).toBe(
      'multilingual',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.max_history_turns')).toBe(
      '1000',
    );
  });

  it('microphone defaults migrate legacy VAD barge-in trigger to microphone scope', () => {
    const defaults = GetDefaultMicrophoneConfig([
      createMetadata('microphone.vad.barge_in_trigger', 'word'),
    ]);

    expect(getMetadataValue(defaults, 'microphone.barge_in_trigger')).toBe(
      'word',
    );
    expect(getMetadataValue(defaults, 'microphone.vad.barge_in_trigger')).toBe(
      '',
    );
  });

  it('noise provider update clears stale denoising params and keeps only provider', () => {
    const seed = [
      createMetadata('listen.model', 'nova-3'),
      createMetadata('microphone.denoising.provider', 'legacy_noise'),
      createMetadata('microphone.denoising.level', 'high'),
    ];

    const current = GetDefaultNoiseCancellationConfig(
      'rn_noise',
      cloneMetadata(seed),
    );
    expect(normalizeMetadata(current)).toEqual(
      normalizeMetadata([
        createMetadata('listen.model', 'nova-3'),
        createMetadata('microphone.denoising.provider', 'rn_noise'),
      ]),
    );
  });

  it('unknown providers are no-op for config-only defaults', () => {
    const seed = [
      createMetadata('listen.model', 'nova-3'),
      createMetadata('microphone.eos.fallback_timeout', '700'),
      createMetadata('microphone.vad.confidence', '0.6'),
      createMetadata('microphone.denoising.provider', 'rn_noise'),
    ];

    expect(
      normalizeMetadata(
        GetDefaultVADConfig('unknown_vad', cloneMetadata(seed)),
      ),
    ).toEqual(normalizeMetadata(seed));
    expect(
      normalizeMetadata(
        GetDefaultEOSConfig('unknown_eos', cloneMetadata(seed)),
      ),
    ).toEqual(normalizeMetadata(seed));
    expect(
      normalizeMetadata(
        GetDefaultNoiseCancellationConfig('unknown_noise', cloneMetadata(seed)),
      ),
    ).toEqual(normalizeMetadata(seed));
  });
});
