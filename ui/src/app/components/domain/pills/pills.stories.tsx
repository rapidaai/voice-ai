import type { Meta, StoryObj } from '@storybook/react-vite';
import { DocumentSourcePill } from './document-source-pill';
import { ProviderPill } from './provider-model-pill';
import { ToolProviderPill } from './tool-provider-pill';

const toolProvider = {
  getId: () => 'tool-1',
  getImage: () => '/logo192.png',
  getName: () => 'Webhook tool',
};

const meta = {
  title: 'Domain/Pills',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Domain pills for providers, tool providers, and document source labels.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Providers: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <ProviderPill provider="openai" />
      <ProviderPill provider="custom-provider" />
    </div>
  ),
};

export const ToolProviders: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <ToolProviderPill toolProvider={toolProvider} />
      <ToolProviderPill toolProviderId="missing-tool" />
    </div>
  ),
};

export const DocumentSources: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <DocumentSourcePill type="manual" />
      <DocumentSourcePill source="upload">Uploaded file</DocumentSourcePill>
      <DocumentSourcePill type="tool" source="missing-tool" />
    </div>
  ),
};
