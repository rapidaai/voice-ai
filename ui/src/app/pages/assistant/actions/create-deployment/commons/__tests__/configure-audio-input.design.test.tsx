import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { Metadata } from '@rapidaai/react';
import { ConfigureAudioInputProvider } from '../configure-audio-input';

const mockGetDefaultMicrophoneConfig = jest.fn();
const mockGetDefaultSpeechToTextIfInvalid = jest.fn();
const mockGetDefaultVADConfig = jest.fn();
const mockGetDefaultEOSConfig = jest.fn();
const mockGetDefaultNoiseCancellationConfig = jest.fn();

jest.mock('@/utils', () => ({
  cn: (...inputs: any[]) => inputs.filter(Boolean).join(' '),
}));

jest.mock('lucide-react', () => ({
  ChevronDown: () => null,
}));

jest.mock('@/app/components/blocks/section-divider', () => ({
  SectionDivider: ({ label }: { label: string }) => <div>{label}</div>,
}));

jest.mock('@/app/components/providers/speech-to-text', () => ({
  SpeechToTextProvider: ({
    onChangeProvider,
  }: {
    onChangeProvider: (provider: string) => void;
  }) => (
    <button onClick={() => onChangeProvider('groq')} type="button">
      change stt
    </button>
  ),
}));

jest.mock('@/app/components/providers/vad', () => ({
  VADProvider: ({
    onChangeProvider,
  }: {
    onChangeProvider: (provider: string) => void;
  }) => (
    <button onClick={() => onChangeProvider('firered_vad')} type="button">
      change vad
    </button>
  ),
}));

jest.mock('@/app/components/providers/microphone/barge-in-trigger-control', () => ({
  MICROPHONE_BARGE_IN_TRIGGER_KEY: 'microphone.barge_in_trigger',
  BargeInTriggerControl: () => <div>barge-in control</div>,
}));

jest.mock('@/app/components/providers/end-of-speech', () => ({
  EndOfSpeechProvider: ({
    onChangeProvider,
  }: {
    onChangeProvider: (provider: string) => void;
  }) => (
    <button onClick={() => onChangeProvider('livekit_eos')} type="button">
      change eos
    </button>
  ),
}));

jest.mock('@/app/components/providers/noise-removal', () => ({
  NoiseCancellationProvider: ({
    onChangeNoiseCancellationProvider,
  }: {
    onChangeNoiseCancellationProvider: (provider: string) => void;
  }) => (
    <button
      onClick={() => onChangeNoiseCancellationProvider('rn_noise')}
      type="button"
    >
      change noise
    </button>
  ),
}));

jest.mock('@/app/components/providers/speech-to-text/provider', () => ({
  GetDefaultMicrophoneConfig: (...args: any[]) =>
    mockGetDefaultMicrophoneConfig(...args),
  GetDefaultSpeechToTextIfInvalid: (...args: any[]) =>
    mockGetDefaultSpeechToTextIfInvalid(...args),
}));

jest.mock('@/app/components/providers/vad/provider', () => ({
  GetDefaultVADConfig: (...args: any[]) => mockGetDefaultVADConfig(...args),
}));

jest.mock('@/app/components/providers/end-of-speech/provider', () => ({
  GetDefaultEOSConfig: (...args: any[]) => mockGetDefaultEOSConfig(...args),
}));

jest.mock('@/app/components/providers/noise-removal/provider', () => ({
  GetDefaultNoiseCancellationConfig: (...args: any[]) =>
    mockGetDefaultNoiseCancellationConfig(...args),
}));

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

describe('ConfigureAudioInputProvider design integration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('uses microphone-only params when switching STT provider', () => {
    const inputParameters = [
      createMetadata('listen.model', 'nova-3'),
      createMetadata('microphone.eos.fallback_timeout', '900'),
      createMetadata('microphone.barge_in_trigger', 'word'),
      createMetadata('microphone.vad.confidence', '0.7'),
      createMetadata('microphone.denoising.provider', 'rn_noise'),
      createMetadata('speaker.model', 'sonic'),
    ];
    const microphoneDefaults = [
      createMetadata('microphone.eos.fallback_timeout', '700'),
      createMetadata('microphone.vad.confidence', '0.6'),
      createMetadata('microphone.denoising.provider', 'rn_noise'),
    ];
    const sttDefaults = [createMetadata('listen.model', 'whisper-large-v3-turbo')];

    mockGetDefaultMicrophoneConfig.mockReturnValue(microphoneDefaults);
    mockGetDefaultSpeechToTextIfInvalid.mockReturnValue(sttDefaults);

    const setAudioInputConfig = jest.fn();
    render(
      <ConfigureAudioInputProvider
        audioInputConfig={{ provider: 'deepgram', parameters: inputParameters }}
        setAudioInputConfig={setAudioInputConfig}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'change stt' }));

    expect(mockGetDefaultMicrophoneConfig).toHaveBeenCalledTimes(1);
    const microphoneOnly = mockGetDefaultMicrophoneConfig.mock.calls[0][0] as Metadata[];
    expect(microphoneOnly.map(m => m.getKey()).sort()).toEqual(
      [
        'microphone.eos.fallback_timeout',
        'microphone.barge_in_trigger',
        'microphone.vad.confidence',
        'microphone.denoising.provider',
      ].sort(),
    );

    expect(mockGetDefaultSpeechToTextIfInvalid).toHaveBeenCalledWith(
      'groq',
      microphoneDefaults,
    );
    expect(setAudioInputConfig).toHaveBeenCalledWith({
      provider: 'groq',
      parameters: sttDefaults,
    });
  });

  it('advanced VAD/EOS/noise controls call the expected defaulting functions', () => {
    const inputParameters = [
      createMetadata('listen.model', 'nova-3'),
      createMetadata('microphone.vad.provider', 'silero_vad'),
      createMetadata('microphone.eos.provider', 'silence_based_eos'),
      createMetadata('microphone.denoising.provider', 'legacy_noise'),
    ];
    const vadDefaults = [createMetadata('microphone.vad.provider', 'firered_vad')];
    const eosDefaults = [createMetadata('microphone.eos.provider', 'livekit_eos')];
    const noiseDefaults = [createMetadata('microphone.denoising.provider', 'rn_noise')];

    mockGetDefaultVADConfig.mockReturnValue(vadDefaults);
    mockGetDefaultEOSConfig.mockReturnValue(eosDefaults);
    mockGetDefaultNoiseCancellationConfig.mockReturnValue(noiseDefaults);

    const setAudioInputConfig = jest.fn();
    render(
      <ConfigureAudioInputProvider
        audioInputConfig={{ provider: 'deepgram', parameters: inputParameters }}
        setAudioInputConfig={setAudioInputConfig}
      />,
    );

    fireEvent.click(
      screen.getByRole('button', { name: /show advanced settings/i }),
    );
    fireEvent.click(screen.getByRole('button', { name: 'change vad' }));
    fireEvent.click(screen.getByRole('button', { name: 'change noise' }));
    fireEvent.click(screen.getByRole('button', { name: 'change eos' }));

    expect(mockGetDefaultVADConfig).toHaveBeenCalledWith(
      'firered_vad',
      inputParameters,
    );
    expect(mockGetDefaultNoiseCancellationConfig).toHaveBeenCalledWith(
      'rn_noise',
      inputParameters,
    );
    expect(mockGetDefaultEOSConfig).toHaveBeenCalledWith(
      'livekit_eos',
      [
        createMetadata('listen.model', 'nova-3'),
        createMetadata('microphone.vad.provider', 'silero_vad'),
        createMetadata('microphone.denoising.provider', 'legacy_noise'),
      ],
    );

    expect(setAudioInputConfig).toHaveBeenCalledWith({
      provider: 'deepgram',
      parameters: vadDefaults,
    });
    expect(setAudioInputConfig).toHaveBeenCalledWith({
      provider: 'deepgram',
      parameters: noiseDefaults,
    });
    expect(setAudioInputConfig).toHaveBeenCalledWith({
      provider: 'deepgram',
      parameters: eosDefaults,
    });
  });

  it('toggles advanced STT settings visibility', () => {
    render(
      <ConfigureAudioInputProvider
        audioInputConfig={{ provider: 'deepgram', parameters: [] }}
        setAudioInputConfig={jest.fn()}
      />,
    );

    expect(screen.queryByRole('button', { name: 'change vad' })).toBeNull();

    fireEvent.click(
      screen.getByRole('button', { name: /show advanced settings/i }),
    );
    expect(screen.getByRole('button', { name: 'change vad' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'change eos' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'change noise' })).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole('button', { name: /hide advanced settings/i }),
    );
    expect(screen.queryByRole('button', { name: 'change vad' })).toBeNull();
  });
});
