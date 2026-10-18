import type { Meta, StoryObj } from '@storybook/react-vite';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext } from '@/context/auth-context';
import { ClickableKnowledgeCard, SelectKnowledgeCard } from './knowledge-card';
import { ProviderCard } from './provider-card';
import { SelectToolCard } from './tool-card';

const authContextValue = {
  currentProjectRole: { projectid: 'project-1' },
  currentUser: { id: 'user-1' },
  organizationRole: { organizationid: 'org-1' },
  token: { token: 'storybook-token' },
};

const knowledge = {
  getDescription: () => 'Support policy, billing, and product documentation.',
  getDocumentcount: () => 12,
  getId: () => 'knowledge-1',
  getName: () => 'Support knowledge',
  getTokencount: () => 34567,
  getWordcount: () => 12345,
};

const provider = {
  code: 'openai',
  description: 'OpenAI model provider for language and reasoning models.',
  featureList: ['external'],
  image: '/logo192.png',
  name: 'OpenAI',
  url: '/providers/openai',
};

const tool = {
  getDescription: () => 'Looks up customer account details from an MCP server.',
  getExecutionmethod: () => 'mcp',
  getExecutionoptionsList: () => [],
  getName: () => 'Customer lookup',
};

const meta = {
  title: 'Domain/Cards',
  tags: ['autodocs'],
  decorators: [
    Story => (
      <MemoryRouter>
        <AuthContext.Provider value={authContextValue as never}>
          <Story />
        </AuthContext.Provider>
      </MemoryRouter>
    ),
  ],
  parameters: {
    docs: {
      description: {
        component:
          'Domain cards for knowledge bases, providers, and assistant tools.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const KnowledgeCards: Story = {
  render: () => (
    <div className="grid max-w-4xl gap-4 md:grid-cols-2">
      <SelectKnowledgeCard
        knowledge={knowledge as never}
        knowledgeOptions={[
          {
            option: 'Archive',
            onActionClick: () => undefined,
          },
        ]}
      />
      <ClickableKnowledgeCard knowledge={knowledge as never} />
    </div>
  ),
};

export const Provider: Story = {
  render: () => (
    <div className="max-w-sm">
      <ProviderCard provider={provider} />
    </div>
  ),
};

export const Tool: Story = {
  render: () => (
    <div className="max-w-sm">
      <SelectToolCard
        tool={tool as never}
        onDelete={() => undefined}
        onEdit={() => undefined}
      />
    </div>
  ),
};
