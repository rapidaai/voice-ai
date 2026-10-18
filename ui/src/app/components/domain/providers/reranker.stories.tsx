import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import { RerankerProvider } from './reranker';

const meta = {
  title: 'Domain/Providers/Reranker',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Reranker provider selector and model configuration used by knowledge retrieval ranking setup.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const RerankerSetup: Story = {
  render: function Render() {
    const [provider, setProvider] = useState('');
    const [parameters, setParameters] = useState<Metadata[]>([]);

    return (
      <Stack gap={7} className="max-w-3xl">
        <RerankerProvider
          provider={provider}
          onChangeProvider={setProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
      </Stack>
    );
  },
};
