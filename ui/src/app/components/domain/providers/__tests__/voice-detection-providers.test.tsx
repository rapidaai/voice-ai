import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { EndOfSpeechProvider } from '../end-of-speech';
import {
  EndOfSpeechConfigComponent,
  GetDefaultEOSConfig,
} from '../end-of-speech/provider';
import { NoiseCancellationProvider } from '../noise-removal';
import {
  GetDefaultNoiseCancellationConfig,
  NoiseCancellationConfigComponent,
} from '../noise-removal/provider';
import { VADProvider } from '../vad';
import { GetDefaultVADConfig, VADConfigComponent } from '../vad/provider';
import userEvent from '@testing-library/user-event';

const mockGetDefaultsFromConfig = jest.fn();
const mockLoadProviderConfig = jest.fn();

jest.mock('@/providers', () => ({
  EndOfSpeech: () => [
    {
      code: 'livekit_eos',
      featureList: ['end_of_speech'],
      name: 'LiveKit EOS',
    },
  ],
  NoiseCancellation: () => [
    {
      code: 'rn_noise',
      featureList: ['noise_cancellation'],
      name: 'RNNoise',
    },
  ],
  VAD: () => [
    {
      code: 'silero_vad',
      featureList: ['vad'],
      name: 'Silero VAD',
    },
  ],
}));

jest.mock('@/providers/config-loader', () => ({
  loadProviderConfig: (...args: unknown[]) => mockLoadProviderConfig(...args),
}));

jest.mock('@/providers/config-defaults', () => ({
  getDefaultsFromConfig: (...args: unknown[]) =>
    mockGetDefaultsFromConfig(...args),
}));

jest.mock('@/app/components/domain/providers/config-renderer', () => ({
  ConfigRenderer: ({ category, provider }: any) => (
    <div data-testid={`config-${category}`}>{provider}</div>
  ),
}));

jest.mock('@/app/components/ui/primitives', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@carbon/react', () => ({
  Dropdown: ({
    id,
    label,
    items = [],
    selectedItem,
    itemToString,
    onChange,
  }: any) => (
    <div>
      <span data-testid={`${id}-empty-name`}>{itemToString?.(null)}</span>
      <select
        aria-label={label}
        value={selectedItem?.code ?? ''}
        onChange={event => {
          const selected = items.find(
            (item: any) => item.code === event.target.value,
          );
          onChange?.({ selectedItem: selected ?? null });
        }}
      >
        <option value="">Select</option>
        {items.map((item: any) => (
          <option key={item.code} value={item.code}>
            {itemToString(item)}
          </option>
        ))}
      </select>
    </div>
  ),
}));

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

const getMetadataValue = (parameters: Metadata[], key: string) =>
  parameters.find(parameter => parameter.getKey() === key)?.getValue();

beforeEach(() => {
  mockGetDefaultsFromConfig.mockReset();
  mockLoadProviderConfig.mockReset();
  mockGetDefaultsFromConfig.mockImplementation(
    (_config, _category, current: Metadata[]) => current,
  );
});

describe('voice detection provider selectors', () => {
  it('renders VAD selector and config only after provider selection', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();
    mockLoadProviderConfig.mockReturnValue({ vad: { parameters: [] } });

    const { rerender } = render(
      <VADProvider
        provider=""
        onChangeProvider={onChangeProvider}
        parameters={[]}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.queryByTestId('config-vad')).not.toBeInTheDocument();
    expect(screen.getByTestId('vad-provider-empty-name')).toBeEmptyDOMElement();

    const select = screen.getByRole('combobox', {
      name: 'Select VAD provider',
    });
    await user.selectOptions(select, String('silero_vad'));
    await user.selectOptions(select, String(''));

    expect(onChangeProvider).toHaveBeenCalledWith('silero_vad');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);

    rerender(
      <VADProvider
        provider="silero_vad"
        onChangeProvider={onChangeProvider}
        parameters={[]}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.getByTestId('config-vad')).toHaveTextContent('silero_vad');
  });

  it('renders end-of-speech selector and config only after provider selection', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();
    mockLoadProviderConfig.mockReturnValue({ eos: { parameters: [] } });

    const { rerender } = render(
      <EndOfSpeechProvider
        provider=""
        onChangeProvider={onChangeProvider}
        parameters={[]}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.queryByTestId('config-eos')).not.toBeInTheDocument();

    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Select end of speech provider' }),
      String('livekit_eos'),
    );
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Select end of speech provider' }),
      String(''),
    );

    expect(onChangeProvider).toHaveBeenCalledWith('livekit_eos');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);

    rerender(
      <EndOfSpeechProvider
        provider="livekit_eos"
        onChangeProvider={onChangeProvider}
        parameters={[]}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.getByTestId('config-eos')).toHaveTextContent('livekit_eos');
  });

  it('renders noise cancellation selector and requires parameters for config', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();
    const onChangeParameter = jest.fn();
    mockLoadProviderConfig.mockReturnValue({
      noise: { parameters: [{ key: 'level' }] },
    });

    const { rerender } = render(
      <NoiseCancellationProvider
        noiseCancellationProvider="rn_noise"
        onChangeNoiseCancellationProvider={onChangeProvider}
      />,
    );

    expect(screen.queryByTestId('config-noise')).not.toBeInTheDocument();

    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Select noise removal provider' }),
      String('rn_noise'),
    );
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Select noise removal provider' }),
      String(''),
    );

    expect(onChangeProvider).toHaveBeenCalledWith('rn_noise');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);

    rerender(
      <NoiseCancellationProvider
        noiseCancellationProvider="rn_noise"
        onChangeNoiseCancellationProvider={onChangeProvider}
        parameters={[]}
      />,
    );
    expect(screen.queryByTestId('config-noise')).not.toBeInTheDocument();

    rerender(
      <NoiseCancellationProvider
        noiseCancellationProvider=""
        onChangeNoiseCancellationProvider={onChangeProvider}
        parameters={[]}
        onChangeParameter={onChangeParameter}
      />,
    );
    expect(screen.queryByTestId('config-noise')).not.toBeInTheDocument();

    rerender(
      <NoiseCancellationProvider
        noiseCancellationProvider="rn_noise"
        onChangeNoiseCancellationProvider={onChangeProvider}
        parameters={[]}
        onChangeParameter={onChangeParameter}
      />,
    );

    expect(screen.getByTestId('config-noise')).toHaveTextContent('rn_noise');
  });
});

describe('voice detection provider defaults and config renderers', () => {
  it('keeps current values when a VAD provider has no config', () => {
    const current = [createMetadata('custom.key', 'custom')];
    mockLoadProviderConfig.mockReturnValue({});

    expect(GetDefaultVADConfig('unknown', current)).toBe(current);
  });

  it('replaces scoped VAD values and preserves unrelated metadata', () => {
    const current = [
      createMetadata('custom.key', 'custom'),
      createMetadata('microphone.vad.provider', 'old_vad'),
      createMetadata('microphone.vad.confidence', '0.2'),
    ];
    mockLoadProviderConfig.mockReturnValue({ vad: { parameters: [] } });
    mockGetDefaultsFromConfig.mockReturnValue(current);

    const result = GetDefaultVADConfig('silero_vad', current);

    expect(getMetadataValue(result, 'custom.key')).toBe('custom');
    expect(getMetadataValue(result, 'microphone.vad.provider')).toBe(
      'silero_vad',
    );
    expect(getMetadataValue(result, 'microphone.vad.confidence')).toBe('0.2');
  });

  it('renders VAD config only when provider config exists', () => {
    const props = {
      provider: 'silero_vad',
      onChangeProvider: () => undefined,
      parameters: [],
      onChangeParameter: () => undefined,
    };
    mockLoadProviderConfig.mockReturnValue({});

    const { rerender } = render(<VADConfigComponent {...props} />);
    expect(screen.queryByTestId('config-vad')).not.toBeInTheDocument();

    mockLoadProviderConfig.mockReturnValue({ vad: { parameters: [] } });
    rerender(<VADConfigComponent {...props} />);

    expect(screen.getByTestId('config-vad')).toHaveTextContent('silero_vad');
  });

  it('keeps current values when an end-of-speech provider has no config', () => {
    const current = [createMetadata('custom.key', 'custom')];
    mockLoadProviderConfig.mockReturnValue({});

    expect(GetDefaultEOSConfig('unknown', current)).toBe(current);
  });

  it('replaces scoped end-of-speech values and preserves unrelated metadata', () => {
    const current = [
      createMetadata('custom.key', 'custom'),
      createMetadata('microphone.eos.provider', 'old_eos'),
      createMetadata('microphone.eos.threshold', '0.1'),
    ];
    mockLoadProviderConfig.mockReturnValue({ eos: { parameters: [] } });
    mockGetDefaultsFromConfig.mockReturnValue(current);

    const result = GetDefaultEOSConfig('livekit_eos', current);

    expect(getMetadataValue(result, 'custom.key')).toBe('custom');
    expect(getMetadataValue(result, 'microphone.eos.provider')).toBe(
      'livekit_eos',
    );
    expect(getMetadataValue(result, 'microphone.eos.threshold')).toBe('0.1');
  });

  it('renders end-of-speech config only when provider config exists', () => {
    const props = {
      provider: 'livekit_eos',
      onChangeProvider: () => undefined,
      parameters: [],
      onChangeParameter: () => undefined,
    };
    mockLoadProviderConfig.mockReturnValue({});

    const { rerender } = render(<EndOfSpeechConfigComponent {...props} />);
    expect(screen.queryByTestId('config-eos')).not.toBeInTheDocument();

    mockLoadProviderConfig.mockReturnValue({ eos: { parameters: [] } });
    rerender(<EndOfSpeechConfigComponent {...props} />);

    expect(screen.getByTestId('config-eos')).toHaveTextContent('livekit_eos');
  });

  it('keeps current values when a noise provider has no config', () => {
    const current = [createMetadata('custom.key', 'custom')];
    mockLoadProviderConfig.mockReturnValue({});

    expect(GetDefaultNoiseCancellationConfig('unknown', current)).toBe(current);
  });

  it('appends noise provider metadata when defaults do not include it', () => {
    const current = [createMetadata('custom.key', 'custom')];
    mockLoadProviderConfig.mockReturnValue({ noise: { parameters: [] } });
    mockGetDefaultsFromConfig.mockReturnValue(current);

    const result = GetDefaultNoiseCancellationConfig('rn_noise', current);

    expect(getMetadataValue(result, 'custom.key')).toBe('custom');
    expect(getMetadataValue(result, 'microphone.denoising.provider')).toBe(
      'rn_noise',
    );
  });

  it('replaces existing noise provider metadata when defaults include it', () => {
    const current = [createMetadata('microphone.denoising.provider', 'old')];
    mockLoadProviderConfig.mockReturnValue({ noise: { parameters: [] } });
    mockGetDefaultsFromConfig.mockReturnValue(current);

    const result = GetDefaultNoiseCancellationConfig('rn_noise', current);

    expect(getMetadataValue(result, 'microphone.denoising.provider')).toBe(
      'rn_noise',
    );
  });

  it('renders noise config only when provider config has parameters', () => {
    const props = {
      provider: 'rn_noise',
      onChangeProvider: () => undefined,
      parameters: [],
      onChangeParameter: () => undefined,
    };
    mockLoadProviderConfig.mockReturnValue({});

    const { rerender } = render(
      <NoiseCancellationConfigComponent {...props} />,
    );
    expect(screen.queryByTestId('config-noise')).not.toBeInTheDocument();

    mockLoadProviderConfig.mockReturnValue({ noise: { parameters: [] } });
    rerender(<NoiseCancellationConfigComponent {...props} />);
    expect(screen.queryByTestId('config-noise')).not.toBeInTheDocument();

    mockLoadProviderConfig.mockReturnValue({
      noise: { parameters: [{ key: 'level' }] },
    });
    rerender(<NoiseCancellationConfigComponent {...props} />);

    expect(screen.getByTestId('config-noise')).toHaveTextContent('rn_noise');
  });
});
