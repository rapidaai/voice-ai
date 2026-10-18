import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import { TelemetryProvider } from './telemetry';

const meta = {
  title: 'Domain/Providers/Telemetry',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Telemetry provider selector and credential configuration used by assistant observability setup.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const TelemetrySetup: Story = {
  render: function Render() {
    const [provider, setProvider] = useState('');
    const [parameters, setParameters] = useState<Metadata[]>([]);

    return (
      <Stack gap={7} className="max-w-3xl">
        <TelemetryProvider
          provider={provider}
          onChangeProvider={setProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
      </Stack>
    );
  },
};
