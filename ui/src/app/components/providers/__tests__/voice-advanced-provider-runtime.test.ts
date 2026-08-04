import { Metadata } from '@rapidaai/react';
import { EndOfSpeech, NoiseCancellation, VAD } from '@/providers';
import { loadProviderConfig } from '@/providers/config-loader';
import { GetDefaultVADConfig } from '@/app/components/providers/vad/provider';
import { GetDefaultEOSConfig } from '@/app/components/providers/end-of-speech/provider';
import { GetDefaultNoiseCancellationConfig } from '@/app/components/providers/noise-removal/provider';

const meta = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

const getValue = (list: Metadata[], key: string): string | undefined =>
  list.find(item => item.getKey() === key)?.getValue();

describe('Voice advanced provider runtime parity', () => {
  it('VAD providers are config-driven and include selected provider key in defaults', () => {
    const configured = VAD().filter(provider =>
      Boolean(loadProviderConfig(provider.code)?.vad),
    );
    expect(configured.length).toBeGreaterThan(0);

    for (const provider of configured) {
      const defaults = GetDefaultVADConfig(provider.code, [
        meta('listen.model', 'nova-3'),
      ]);
      expect(getValue(defaults, 'microphone.vad.provider')).toBe(provider.code);
      expect(getValue(defaults, 'microphone.vad.barge_in_trigger')).toBe('vad');
      expect(getValue(defaults, 'listen.model')).toBe('nova-3');
    }
  });

  it('EOS providers are config-driven and include selected provider key in defaults', () => {
    const configured = EndOfSpeech().filter(provider =>
      Boolean(loadProviderConfig(provider.code)?.eos),
    );
    expect(configured.length).toBeGreaterThan(0);

    for (const provider of configured) {
      const defaults = GetDefaultEOSConfig(provider.code, [
        meta('listen.model', 'nova-3'),
      ]);
      expect(getValue(defaults, 'microphone.eos.provider')).toBe(provider.code);
      expect(getValue(defaults, 'listen.model')).toBe('nova-3');
    }
  });

  it('Noise providers are config-driven and include selected provider key in defaults', () => {
    const configured = NoiseCancellation().filter(provider =>
      Boolean(loadProviderConfig(provider.code)?.noise),
    );
    expect(configured.length).toBeGreaterThan(0);

    for (const provider of configured) {
      const defaults = GetDefaultNoiseCancellationConfig(provider.code, [
        meta('listen.model', 'nova-3'),
      ]);
      expect(getValue(defaults, 'microphone.denoising.provider')).toBe(
        provider.code,
      );
      expect(getValue(defaults, 'listen.model')).toBe('nova-3');
    }
  });

  it('unknown providers are no-op for vad/eos/noise defaults', () => {
    const seed = [meta('custom.key', 'value')];
    expect(GetDefaultVADConfig('unknown-provider', seed)).toBe(seed);
    expect(GetDefaultEOSConfig('unknown-provider', seed)).toBe(seed);
    expect(GetDefaultNoiseCancellationConfig('unknown-provider', seed)).toBe(
      seed,
    );
  });
});
