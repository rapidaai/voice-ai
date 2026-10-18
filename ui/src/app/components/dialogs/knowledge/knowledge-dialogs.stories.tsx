import type { Meta, StoryObj } from '@storybook/react-vite';
import { AuthContext } from '@/context/auth-context';
import {
  CreateKnowledgeDocumentDialog,
  DeleteKnowledgeDocumentSegmentDialog,
  EditKnowledgeDocumentSegmentDialog,
  HowKnowledgeWorksDialog,
} from '.';

const authContextValue = {
  currentUser: { id: 'user-1' },
  token: { token: 'storybook-token' },
  currentProjectRole: { projectid: 'project-1' },
  organizationRole: { organizationid: 'org-1' },
};

const mockSegment = {
  getDocumentId: () => 'document-1',
  getIndex: () => 12,
  getMetadata: () => ({
    getDocumentName: () => 'Support guide',
  }),
  getEntities: () => ({
    getOrganizationsList: () => ['Acme'],
    getDatesList: () => ['2026-09-20'],
    getProductsList: () => ['Voice AI'],
    getEventsList: () => ['Launch'],
    getIndustriesList: () => ['Customer support'],
    getLocationsList: () => ['US'],
    getPeopleList: () => ['Priyanka'],
    getTimesList: () => ['09:00'],
    getQuantitiesList: () => ['3'],
  }),
};

const meta = {
  title: 'Dialogs/Knowledge',
  tags: ['autodocs'],
  decorators: [
    Story => (
      <AuthContext.Provider value={authContextValue as never}>
        <Story />
      </AuthContext.Provider>
    ),
  ],
  parameters: {
    docs: {
      description: {
        component:
          'Knowledge dialogs for document upload, segment editing, segment deletion, and workflow education.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const CreateDocument: Story = {
  render: () => (
    <CreateKnowledgeDocumentDialog
      modalOpen
      setModalOpen={() => undefined}
      knowledgeId="knowledge-1"
      onReload={() => undefined}
    />
  ),
};

export const EditSegment: Story = {
  render: () => (
    <EditKnowledgeDocumentSegmentDialog
      segment={mockSegment as never}
      onClose={() => undefined}
      onUpdate={() => undefined}
    />
  ),
};

export const DeleteSegment: Story = {
  render: () => (
    <DeleteKnowledgeDocumentSegmentDialog
      segment={mockSegment as never}
      onClose={() => undefined}
      onDelete={() => undefined}
    />
  ),
};

export const HowKnowledgeWorks: Story = {
  render: () => (
    <HowKnowledgeWorksDialog modalOpen setModalOpen={() => undefined} />
  ),
};
