import type { Meta, StoryObj } from '@storybook/react-vite';
import { Add, Search } from '@carbon/icons-react';
import { Button } from '@carbon/react';
import { EmptyState } from './empty-state';

const meta = {
  title: 'UI/Feedback/EmptyState',
  component: EmptyState,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Reusable empty state for tables, lists, and panels. Uses Carbon tokens and Carbon Button actions.',
      },
    },
  },
  args: {
    icon: Search,
    title: 'No records found',
    subtitle: 'Adjust filters or create a new record to populate this view.',
    action: 'Create record',
    actionIcon: Add,
    onAction: () => undefined,
  },
} satisfies Meta<typeof EmptyState>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithCustomAction: Story = {
  args: {
    action: undefined,
    actionIcon: undefined,
    actionComponent: (
      <Button kind="secondary" size="md">
        Review filters
      </Button>
    ),
  },
};
