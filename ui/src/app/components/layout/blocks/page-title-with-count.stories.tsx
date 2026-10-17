import type { Meta, StoryObj } from '@storybook/react-vite';
import { PageTitleWithCount } from './page-title-with-count';

const meta = {
  title: 'Layout/Blocks/PageTitleWithCount',
  component: PageTitleWithCount,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Page title variant that shows visible result count and total count.',
      },
    },
  },
  args: {
    count: 12,
    total: 42,
    children: 'Assistants',
  },
} satisfies Meta<typeof PageTitleWithCount>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Empty: Story = {
  args: {
    count: 0,
    total: 0,
    children: 'Projects',
  },
};
