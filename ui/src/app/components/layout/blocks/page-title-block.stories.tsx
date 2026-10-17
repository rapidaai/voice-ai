import type { Meta, StoryObj } from '@storybook/react-vite';
import { PageTitleBlock } from './page-title-block';

const meta = {
  title: 'Layout/Blocks/PageTitleBlock',
  component: PageTitleBlock,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component: 'Compact page title text used inside page header rows.',
      },
    },
  },
  args: {
    children: 'Assistants',
  },
} satisfies Meta<typeof PageTitleBlock>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
