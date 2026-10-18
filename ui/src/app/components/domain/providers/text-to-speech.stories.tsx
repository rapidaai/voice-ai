import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import { TextToSpeechProvider } from './text-to-speech';

const meta = {
  title: 'Domain/Providers/Text To Speech',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Text-to-speech provider selector and credential configuration used by assistant voice output setup.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const VoiceOutputSetup: Story = {
  render: function Render() {
    const [provider, setProvider] = useState('');
    const [parameters, setParameters] = useState<Metadata[]>([]);

    return (
      <Stack gap={7} className="max-w-3xl">
        <TextToSpeechProvider
          provider={provider}
          onChangeProvider={setProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
      </Stack>
    );
  },
};
