import type { Meta, StoryObj } from '@storybook/react-vite';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { AuthContext } from '@/context/auth-context';
import { ActionableHeader } from './actionable-header';
import { breadcrumb } from '@/app/routes/breadcrumb-handle';

const projectRoles = [
  { id: 'project-a', projectid: 'project-a', projectname: 'Project Alpha' },
  { id: 'project-b', projectid: 'project-b', projectname: 'Project Beta' },
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
            projectRoles,
            currentProjectRole: projectRoles[0],
            setCurrentProjectRole: () => undefined,
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
          'Workspace header that renders route breadcrumbs, project selection, theme toggle, and account links.',
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
