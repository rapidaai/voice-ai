import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import { ConfigRenderer } from './config-renderer';
import type { CategoryConfig } from '@/providers/config-loader';

const config: CategoryConfig = {
  parameters: [
    {
      key: 'endpoint.url',
      label: 'Endpoint URL',
      type: 'input',
      placeholder: 'https://api.example.com',
      helpText: 'Provider endpoint used by the runtime.',
    },
    {
      key: 'transport',
      label: 'Transport',
      type: 'select',
      choices: [
        { label: 'WebSocket', value: 'websocket' },
        { label: 'HTTP', value: 'http' },
      ],
    },
    {
      key: 'timeout',
      label: 'Timeout',
      type: 'slider',
      min: 1,
      max: 30,
      step: 1,
      default: '10',
    },
    {
      key: 'headers',
      label: 'Headers',
      type: 'key_value',
      colSpan: 2,
    },
    {
      key: 'payload.template',
      label: 'Payload template',
      type: 'json',
      colSpan: 2,
      placeholder: '{"message":"{{input}}"}',
    },
  ],
};

const meta = {
  title: 'Domain/Providers/Config Renderer',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Generic provider configuration renderer for catalog-driven form fields.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const ProviderFields: Story = {
  render: function Render() {
    const [parameters, setParameters] = useState<Metadata[]>([]);

    return (
      <Stack gap={7} className="max-w-3xl">
        <ConfigRenderer
          provider="custom"
          category="telemetry"
          config={config}
          parameters={parameters}
          onParameterChange={setParameters}
        />
      </Stack>
    );
  },
};
