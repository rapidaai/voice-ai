import type { Meta, StoryObj } from '@storybook/react-vite';
import { MemoryRouter } from 'react-router-dom';
import { SidebarProvider } from '@/context/sidebar-context';
import { SidebarNavigation } from './sidebar';

const meta = {
  title: 'Layout/Navigation/SidebarNavigation',
  component: SidebarNavigation,
  tags: ['autodocs'],
  decorators: [
    Story => (
      <MemoryRouter initialEntries={['/deployment/assistant']}>
        <SidebarProvider>
          <div className="h-[520px]">
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
          'Primary workspace sidebar with branded logo, route sections, and collapse control.',
      },
    },
    layout: 'fullscreen',
  },
} satisfies Meta<typeof SidebarNavigation>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
