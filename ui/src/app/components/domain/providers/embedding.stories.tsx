import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import { EmbeddingProvider } from './embedding';

const meta = {
  title: 'Domain/Providers/Embedding',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Embedding provider selector and model configuration used by knowledge retrieval setup.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const EmbeddingSetup: Story = {
  render: function Render() {
    const [provider, setProvider] = useState('');
    const [parameters, setParameters] = useState<Metadata[]>([]);

    return (
      <Stack gap={7} className="max-w-3xl">
        <EmbeddingProvider
          provider={provider}
          onChangeProvider={setProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
      </Stack>
    );
  },
};
