import type { Meta, StoryObj } from '@storybook/react-vite';
import { Dashboard } from '@carbon/icons-react';
import { MemoryRouter } from 'react-router-dom';
import { SidebarProvider } from '@/context/sidebar-context';
import { SidebarIconWrapper } from './sidebar-icon-wrapper';
import { SidebarLabel } from './sidebar-label';
import { SidebarSimpleListItem } from './sidebar-simple-list-item';

const meta = {
  title: 'Layout/Navigation/SidebarSimpleListItem',
  component: SidebarSimpleListItem,
  tags: ['autodocs'],
  decorators: [
    Story => (
      <MemoryRouter initialEntries={['/dashboard']}>
        <SidebarProvider>
          <div
            className="border border-border-subtle bg-shell"
            style={{ width: '13.5rem' }}
          >
            <Story />
          </div>
        </SidebarProvider>
      </MemoryRouter>
    ),
  ],
  parameters: {
    docs: {
      description: {
        component:
          'Sidebar link row used by primary navigation groups and nested navigation items.',
      },
    },
  },
  args: {
    navigate: '/dashboard',
    active: false,
    children: (
      <>
        <SidebarIconWrapper>
          <Dashboard size={16} />
        </SidebarIconWrapper>
        <SidebarLabel>Dashboard</SidebarLabel>
      </>
    ),
  },
} satisfies Meta<typeof SidebarSimpleListItem>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Active: Story = {
  args: {
    active: true,
  },
};

export const Loading: Story = {
  args: {
    loading: true,
  },
};
