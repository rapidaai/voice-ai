import type { Meta, StoryObj } from '@storybook/react-vite';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { AuthContext } from '@/context/auth-context';
import { ActionableHeader } from './actionable-header';
import { breadcrumb } from '@/app/routes/breadcrumb-handle';

const projects = [
  { id: 'project-a', name: 'Project Alpha', rolesList: [] },
  { id: 'project-b', name: 'Project Beta', rolesList: [] },
] as any;
const organizations = [
  {
    id: 'organization-a',
    name: 'Organization Alpha',
    projectsList: projects,
    rolesList: [],
  },
] as any;

const meta = {
  title: 'Layout/Navigation/ActionableHeader',
  component: ActionableHeader,
  tags: ['autodocs'],
  decorators: [
    Story => {
      const router = createMemoryRouter(
        [
          {
            path: 'deployment',
            handle: breadcrumb('Deployment'),
            children: [
              {
                path: 'assistant',
                handle: breadcrumb('Assistants'),
                element: <Story />,
              },
            ],
          },
        ],
        { initialEntries: ['/deployment/assistant'] },
      );

      return (
        <AuthContext.Provider
          value={{
            getOrganizations: () => organizations,
            getCurrentOrganization: () => organizations[0],
            getProjects: () => projects,
            getCurrentProject: () => projects[0],
            setCurrentContext: () => true,
          }}
        >
          <RouterProvider router={router} />
        </AuthContext.Provider>
      );
    },
  ],
  parameters: {
    docs: {
      description: {
        component:
          'Workspace header that renders route breadcrumbs, organization and project selection, theme toggle, and account links.',
      },
    },
  },
} satisfies Meta<typeof ActionableHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const CustomClass: Story = {
  args: {
    className: 'shadow-sm',
  },
};
