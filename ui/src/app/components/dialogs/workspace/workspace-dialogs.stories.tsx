import type { Meta, StoryObj } from '@storybook/react-vite';
import { AuthContext } from '@/context/auth-context';
import {
  CreateProjectDialog,
  InviteOrganizationUserDialog,
  InviteProjectUserDialog,
  UpdateProjectDialog,
} from '.';

const authContextValue = {
  currentUser: { id: 'user-1' },
  token: { token: 'storybook-token' },
  currentProjectRole: { projectid: 'project-1' },
  organizationRole: { organizationid: 'org-1' },
  projectRoles: [
    { projectid: 'project-1', projectname: 'Support operations' },
    { projectid: 'project-2', projectname: 'Sales enablement' },
  ],
  authorize: (onSuccess: () => void) => onSuccess(),
};

const existingProject = {
  id: 'project-1',
  name: 'Support operations',
  description: 'Customer support automation workspace.',
};

const invitedUser = {
  getId: () => 'user-2',
  getName: () => 'Priyanka Iyer',
  getEmail: () => 'p_iyer@outlook.com',
};

const meta = {
  title: 'Dialogs/Workspace',
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
          'Workspace dialogs for project creation, project updates, organization invitations, and project access.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const CreateProject: Story = {
  render: () => (
    <CreateProjectDialog
      modalOpen
      setModalOpen={() => undefined}
      afterCreateProject={() => undefined}
    />
  ),
};

export const UpdateProject: Story = {
  render: () => (
    <UpdateProjectDialog
      modalOpen
      setModalOpen={() => undefined}
      existingProject={existingProject as never}
      afterUpdateProject={() => undefined}
    />
  ),
};

export const InviteOrganizationUser: Story = {
  render: () => (
    <InviteOrganizationUserDialog
      modalOpen
      setModalOpen={() => undefined}
      onSuccess={() => undefined}
    />
  ),
};

export const InviteProjectUser: Story = {
  render: () => (
    <InviteProjectUserDialog
      modalOpen
      setModalOpen={() => undefined}
      user={invitedUser as never}
      projectId="project-1"
      onSuccess={() => undefined}
    />
  ),
};
