import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { CredentialDropdownView } from './credential-dropdown';
import { EndpointDropdownView } from './endpoint-dropdown';
import { KnowledgeDropdownView } from './knowledge-dropdown';

const credentials = [
  {
    getId: () => 'cred-openai',
    getName: () => 'Production OpenAI',
    getProvider: () => 'openai',
  },
  {
    getId: () => 'cred-azure',
    getName: () => 'Azure East US',
    getProvider: () => 'azure',
  },
] as any[];

const endpoints = [
  {
    getId: () => 'endpoint-prod',
    getName: () => 'Production voice endpoint',
  },
  {
    getId: () => 'endpoint-sandbox',
    getName: () => 'Sandbox endpoint',
  },
] as any[];

const knowledgeBases = [
  {
    getId: () => 'kb-support',
    getName: () => 'Support articles',
  },
  {
    getId: () => 'kb-product',
    getName: () => 'Product manuals',
  },
] as any[];

const providerNames = new Map([
  ['openai', 'OpenAI'],
  ['azure', 'Azure'],
]);

const meta = {
  title: 'Domain/Dropdowns',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Carbon dropdown controls for selecting credentials, endpoints, and knowledge bases.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const All: Story = {
  render: () => (
    <Stack gap={5}>
      <CredentialDropdownView
        credentials={credentials as any}
        currentCredential="cred-openai"
        getProviderName={provider => providerNames.get(provider)}
        onChangeCredential={() => undefined}
        onReloadCredentials={() => undefined}
        onCreateCredential={() => undefined}
      />
      <EndpointDropdownView
        endpoints={endpoints as any}
        currentEndpoint="endpoint-prod"
        onChangeEndpoint={() => undefined}
        onRefresh={() => undefined}
        onCreateEndpoint={() => undefined}
      />
      <KnowledgeDropdownView
        knowledgeBases={knowledgeBases as any}
        currentKnowledge="kb-support"
        onChangeKnowledge={() => undefined}
        onRefresh={() => undefined}
        onCreateKnowledge={() => undefined}
      />
    </Stack>
  ),
};

export const Loading: Story = {
  render: () => (
    <Stack gap={5}>
      <EndpointDropdownView
        endpoints={endpoints as any}
        isLoading
        onChangeEndpoint={() => undefined}
        onRefresh={() => undefined}
        onCreateEndpoint={() => undefined}
      />
      <KnowledgeDropdownView
        knowledgeBases={knowledgeBases as any}
        isLoading
        onRefresh={() => undefined}
        onCreateKnowledge={() => undefined}
      />
    </Stack>
  ),
};
