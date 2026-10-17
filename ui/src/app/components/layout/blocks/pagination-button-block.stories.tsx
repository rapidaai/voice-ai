import type { Meta, StoryObj } from '@storybook/react-vite';
import { Button } from '@carbon/react';
import { PaginationButtonBlock } from './pagination-button-block';

const meta = {
  title: 'Layout/Blocks/PaginationButtonBlock',
  component: PaginationButtonBlock,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Inline wrapper for adjacent pagination controls inside compact page headers.',
      },
    },
  },
  args: {
    children: (
      <>
        <Button kind="ghost" size="sm">
          Previous
        </Button>
        <Button kind="ghost" size="sm">
          Next
        </Button>
      </>
    ),
  },
} satisfies Meta<typeof PaginationButtonBlock>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
