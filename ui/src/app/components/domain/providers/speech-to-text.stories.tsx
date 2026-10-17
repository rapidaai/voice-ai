import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import { SpeechToTextProvider } from './speech-to-text';

const meta = {
  title: 'Domain/Providers/Speech To Text',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Speech-to-text provider selector and credential configuration used by assistant voice input setup.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const VoiceInputSetup: Story = {
  render: function Render() {
    const [provider, setProvider] = useState('');
    const [parameters, setParameters] = useState<Metadata[]>([]);

    return (
      <Stack gap={7} className="max-w-3xl">
        <SpeechToTextProvider
          provider={provider}
          onChangeProvider={setProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
      </Stack>
    );
  },
};
