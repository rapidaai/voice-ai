import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import { HelpToggletip } from './help-label';
import {
  BargeInTriggerControl,
  MICROPHONE_BARGE_IN_TRIGGER_KEY,
} from './microphone/barge-in-trigger-control';
import { VADProvider } from './vad';
import { EndOfSpeechProvider } from './end-of-speech';
import { NoiseCancellationProvider } from './noise-removal';

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

const meta = {
  title: 'Domain/Providers/Controls',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Shared controls used by provider configuration panels, including labels and microphone behavior selectors.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const MicrophoneBargeIn: Story = {
  render: function Render() {
    const [parameters, setParameters] = useState<Metadata[]>([
      createMetadata(MICROPHONE_BARGE_IN_TRIGGER_KEY, 'vad'),
    ]);

    return (
      <Stack gap={5} className="max-w-md">
        <BargeInTriggerControl
          parameters={parameters}
          onChangeParameter={setParameters}
        />
        <div className="inline-flex items-center gap-1 text-sm text-gray-600 dark:text-gray-300">
          Shared help label
          <HelpToggletip
            label="Shared help label"
            helpText="Provider controls use Carbon toggletips for compact field guidance."
          />
        </div>
      </Stack>
    );
  },
};

export const AudioDetectionProviders: Story = {
  render: function Render() {
    const [vadProvider, setVADProvider] = useState('silero_vad');
    const [eosProvider, setEOSProvider] = useState('pipecat_smart_turn_eos');
    const [noiseProvider, setNoiseProvider] = useState('rn_noise');
    const [parameters, setParameters] = useState<Metadata[]>([
      createMetadata('microphone.vad.provider', 'silero_vad'),
      createMetadata('microphone.eos.provider', 'pipecat_smart_turn_eos'),
      createMetadata('microphone.denoising.provider', 'rn_noise'),
    ]);

    return (
      <Stack gap={7} className="max-w-3xl">
        <VADProvider
          provider={vadProvider}
          onChangeProvider={setVADProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
        <EndOfSpeechProvider
          provider={eosProvider}
          onChangeProvider={setEOSProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
        <NoiseCancellationProvider
          noiseCancellationProvider={noiseProvider}
          onChangeNoiseCancellationProvider={setNoiseProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
      </Stack>
    );
  },
};
