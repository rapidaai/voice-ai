import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { ProviderTag } from './provider-tag';

const meta = {
  title: 'Domain/Provider Tag',
  component: ProviderTag,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component: 'Carbon tag used to show provider identities consistently.',
      },
    },
  },
} satisfies Meta<typeof ProviderTag>;

export default meta;

type Story = StoryObj<typeof meta>;

export const OpenAI: Story = {
  args: {
    provider: 'openai',
  },
};

export const Gallery: Story = {
  render: () => (
    <Stack gap={3} orientation="horizontal">
      <ProviderTag provider="openai" />
      <ProviderTag provider="azure-openai" />
      <ProviderTag provider="custom-stt" />
      <ProviderTag provider="internal-provider" />
    </Stack>
  ),
};
