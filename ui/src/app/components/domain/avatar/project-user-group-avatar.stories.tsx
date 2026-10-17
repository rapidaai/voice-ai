import type { Meta, StoryObj } from '@storybook/react-vite';
import { ProjectUserGroupAvatar } from './project-user-group-avatar';

const members = [
  { name: 'Priyanka Iyer' },
  { name: 'Amara Shah' },
  { name: 'Diego Rivera' },
  { name: 'Noor Khan' },
];

const meta = {
  title: 'Domain/Avatar/Project User Group',
  component: ProjectUserGroupAvatar,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component: 'Compact overlapping avatar group for project members.',
      },
    },
  },
} satisfies Meta<typeof ProjectUserGroupAvatar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    members,
    size: 8,
  },
};

export const Small: Story = {
  args: {
    members,
    size: 7,
  },
};
