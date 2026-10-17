import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import { TextProvider } from './text';

const meta = {
  title: 'Domain/Providers/Text',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Text model provider selector and credential configuration used by assistant language model setup.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const ModelSetup: Story = {
  render: function Render() {
    const [provider, setProvider] = useState('');
    const [parameters, setParameters] = useState<Metadata[]>([]);

    return (
      <Stack gap={7} className="max-w-4xl">
        <TextProvider
          provider={provider}
          onChangeProvider={setProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
      </Stack>
    );
  },
};
