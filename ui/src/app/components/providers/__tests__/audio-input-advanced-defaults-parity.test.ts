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

const legacyVadDefaults: Record<string, Record<string, string>> = {
  silero_vad: {
    'microphone.vad.barge_in_trigger': 'vad',
    'microphone.vad.threshold': '0.5',
    'microphone.vad.min_silence_frame': '20',
    'microphone.vad.min_speech_frame': '8',
  },
  ten_vad: {
    'microphone.vad.barge_in_trigger': 'vad',
    'microphone.vad.threshold': '0.5',
    'microphone.vad.min_silence_frame': '20',
    'microphone.vad.min_speech_frame': '8',
  },
  firered_vad: {
    'microphone.vad.barge_in_trigger': 'vad',
    'microphone.vad.threshold': '0.5',
    'microphone.vad.min_silence_frame': '10',
    'microphone.vad.min_speech_frame': '3',
  },
};

const legacyEosDefaults: Record<string, Record<string, string>> = {
  silence_based_eos: {
    'microphone.eos.timeout': '700',
  },
  livekit_eos: {
    'microphone.eos.fallback_timeout': '500',
    'microphone.eos.threshold': '0.0289',
    'microphone.eos.quick_timeout': '250',
    'microphone.eos.extended_timeout': '3000',
    'microphone.eos.model': 'en',
  },
  pipecat_smart_turn_eos: {
    'microphone.eos.fallback_timeout': '500',
    'microphone.eos.threshold': '0.5',
    'microphone.eos.quick_timeout': '250',
    'microphone.eos.extended_timeout': '2000',
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

const legacyGetDefaultVADConfig = (
  provider: string,
  current: Metadata[],
): Metadata[] => {
  const defaults = legacyVadDefaults[provider] || {};
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

const legacyGetDefaultEosConfig = (
  provider: string,
  current: Metadata[],
): Metadata[] => {
  const defaults = legacyEosDefaults[provider] || {};
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

  it('all active end-of-speech providers are config-driven', () => {
    expect(EndOfSpeech().length).toBeGreaterThan(0);
    for (const provider of EndOfSpeech()) {
      expect(loadProviderConfig(provider.code)?.eos).toBeDefined();
    }
  });

  it.each(['silero_vad', 'ten_vad', 'firered_vad'])(
    '%s VAD defaults stay parity with legacy behavior',
    provider => {
      const seed = [
        createMetadata('rapida.credential_id', 'cred'),
        createMetadata('listen.model', 'nova-3'),
        createMetadata('microphone.vad.provider', 'ten_vad'),
        createMetadata('microphone.vad.threshold', '0.77'),
        createMetadata('microphone.vad.min_silence_frame', '15'),
        createMetadata('microphone.vad.min_speech_frame', '4'),
      ];

      const legacy = legacyGetDefaultVADConfig(provider, cloneMetadata(seed));
      const current = GetDefaultVADConfig(provider, cloneMetadata(seed));
      expect(normalizeMetadata(current)).toEqual(normalizeMetadata(legacy));
    },
  );

  it.each(['silence_based_eos', 'livekit_eos', 'pipecat_smart_turn_eos'])(
    '%s EOS defaults stay parity with legacy behavior',
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

      const legacy = legacyGetDefaultEosConfig(provider, cloneMetadata(seed));
      const current = GetDefaultEOSConfig(provider, cloneMetadata(seed));
      expect(normalizeMetadata(current)).toEqual(normalizeMetadata(legacy));
    },
  );

  it('VAD provider key is always updated to the selected provider', () => {
    const seed = [
      createMetadata('microphone.vad.provider', 'ten_vad'),
      createMetadata('microphone.vad.threshold', '0.3'),
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
    expect(getMetadataValue(switched, 'microphone.eos.quick_timeout')).toBe(
      '250',
    );
    expect(getMetadataValue(switched, 'microphone.eos.extended_timeout')).toBe(
      '2000',
    );
  });

  it('microphone defaults use pipecat eos provider', () => {
    const defaults = GetDefaultMicrophoneConfig([]);
    expect(getMetadataValue(defaults, 'microphone.vad.barge_in_trigger')).toBe(
      'vad',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.provider')).toBe(
      'pipecat_smart_turn_eos',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.fallback_timeout')).toBe(
      '500',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.threshold')).toBe('0.5');
    expect(getMetadataValue(defaults, 'microphone.eos.quick_timeout')).toBe(
      '250',
    );
    expect(getMetadataValue(defaults, 'microphone.eos.extended_timeout')).toBe(
      '2000',
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
      '500',
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
      createMetadata('microphone.vad.threshold', '0.6'),
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
