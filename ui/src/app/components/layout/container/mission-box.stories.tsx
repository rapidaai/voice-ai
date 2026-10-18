import type { Meta, StoryObj } from '@storybook/react-vite';
import { MemoryRouter } from 'react-router-dom';
import { MissionBox } from './mission-box';

const meta = {
  title: 'Layout/Container/MissionBox',
  component: MissionBox,
  tags: ['autodocs'],
  decorators: [
    Story => (
      <MemoryRouter initialEntries={['/dashboard']}>
        <Story />
      </MemoryRouter>
    ),
  ],
  parameters: {
    docs: {
      description: {
        component:
          'Authenticated app shell container with sidebar, action header, loader, toast, and route content.',
      },
    },
    layout: 'fullscreen',
  },
  args: {
    children: (
      <section className="flex h-full items-center justify-center p-8">
        <p className="text-sm text-muted">Workspace route content</p>
      </section>
    ),
  },
} satisfies Meta<typeof MissionBox>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
