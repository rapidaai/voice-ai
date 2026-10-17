import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import { TelephonyProvider } from './telephony';

const meta = {
  title: 'Domain/Providers/Telephony',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Telephony provider selector and credential configuration used by assistant phone call setup.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const TelephonySetup: Story = {
  render: function Render() {
    const [provider, setProvider] = useState('');
    const [parameters, setParameters] = useState<Metadata[]>([]);

    return (
      <Stack gap={7} className="max-w-3xl">
        <TelephonyProvider
          provider={provider}
          onChangeProvider={setProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
      </Stack>
    );
  },
};
