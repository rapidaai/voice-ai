import type { Meta, StoryObj } from '@storybook/react-vite';
import { InlineNotification, Stack } from '@carbon/react';
import { EndpointIntegration } from './endpoint-integration';

const makeVariable = (name: string, type: string) => ({
  getName: () => name,
  getType: () => type,
});

const endpoint = {
  getEndpointprovidermodel: () => ({
    getChatcompleteprompt: () => ({
      getPromptvariablesList: () => [
        makeVariable('customerName', 'text'),
        makeVariable('landingUrl', 'url'),
        makeVariable('attachment', 'files'),
        makeVariable('voiceAudio', 'audio-files'),
      ],
    }),
    getEndpointid: () => 'support-router',
    getId: () => 'version-42',
  }),
};

const meta = {
  title: 'Domain/Integration Document/Endpoint Integration',
  component: EndpointIntegration as any,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Generated SDK integration guide for an endpoint, including install, authentication, client setup, and invocation snippets.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const SDKGuide: Story = {
  render: () => (
    <Stack gap={5} className="max-w-5xl">
      <EndpointIntegration
        endpoint={endpoint as any}
        credentialCard={
          <InlineNotification
            hideCloseButton
            kind="info"
            lowContrast
            subtitle="Storybook uses a static credential placeholder."
            title="Publishable key"
          />
        }
      />
    </Stack>
  ),
};
