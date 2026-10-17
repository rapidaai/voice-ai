import type { Meta, StoryObj } from '@storybook/react-vite';
import { Header } from './header';

const meta = {
  title: 'Layout/Navigation/Header',
  component: Header,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Application header surface that renders the active brand logo and exposes a route-level banner landmark.',
      },
    },
  },
  args: {
    'aria-label': undefined,
  },
} satisfies Meta<typeof Header>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const CustomLabel: Story = {
  args: {
    'aria-label': 'Workspace navigation',
  },
};
