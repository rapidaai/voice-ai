import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import {
  BuildinTool,
  GetDefaultToolConfigIfInvalid,
  GetDefaultToolDefintion,
  type BuildinToolConfig,
  type ToolDefinition,
} from './tool-registry';

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

const meta = {
  title: 'Domain/Tools/Tool Configuration',
  component: BuildinTool as any,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Built-in assistant tool configuration with source conditions, action selection, parameters, and tool definitions.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const ApiRequestTool: Story = {
  render: function Render() {
    const [config, setConfig] = useState<BuildinToolConfig>({
      code: 'api_request',
      parameters: GetDefaultToolConfigIfInvalid('api_request', [
        createMetadata('tool.method', 'POST'),
        createMetadata('tool.endpoint', 'https://api.example.com/orders'),
      ]),
    });
    const [definition, setDefinition] = useState<ToolDefinition>(
      GetDefaultToolDefintion('api_request'),
    );

    const handleToolChange = (code: string) => {
      setConfig({
        code,
        parameters: GetDefaultToolConfigIfInvalid(code, []),
      });
      setDefinition(GetDefaultToolDefintion(code));
    };

    return (
      <Stack gap={7} className="max-w-5xl">
        <BuildinTool
          toolDefinition={definition}
          onChangeToolDefinition={setDefinition}
          onChangeBuildinTool={handleToolChange}
          config={config}
          onChangeConfig={setConfig}
        />
      </Stack>
    );
  },
};

export const TransferCallTool: Story = {
  render: function Render() {
    const [config, setConfig] = useState<BuildinToolConfig>({
      code: 'transfer_call',
      parameters: GetDefaultToolConfigIfInvalid('transfer_call', [
        createMetadata('tool.transfer_to', '+14155551234'),
        createMetadata('tool.transfer_message', 'Connecting you now.'),
      ]),
    });
    const [definition, setDefinition] = useState<ToolDefinition>(
      GetDefaultToolDefintion('transfer_call'),
    );

    const handleToolChange = (code: string) => {
      setConfig({
        code,
        parameters: GetDefaultToolConfigIfInvalid(code, []),
      });
      setDefinition(GetDefaultToolDefintion(code));
    };

    return (
      <Stack gap={7} className="max-w-5xl">
        <BuildinTool
          toolDefinition={definition}
          onChangeToolDefinition={setDefinition}
          onChangeBuildinTool={handleToolChange}
          config={config}
          onChangeConfig={setConfig}
        />
      </Stack>
    );
  },
};
