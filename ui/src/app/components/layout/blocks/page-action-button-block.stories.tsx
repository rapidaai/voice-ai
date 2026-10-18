import type { Meta, StoryObj } from '@storybook/react-vite';
import { Button } from '@carbon/react';
import { PageActionButtonBlock } from './page-action-button-block';

const meta = {
  title: 'Layout/Blocks/PageActionButtonBlock',
  component: PageActionButtonBlock,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Sticky page action area that can show a validation notice above primary and secondary actions.',
      },
    },
  },
  args: {
    errorMessage: undefined,
    children: (
      <>
        <Button kind="primary" size="sm">
          Save
        </Button>
        <Button kind="secondary" size="sm">
          Cancel
        </Button>
      </>
    ),
  },
} satisfies Meta<typeof PageActionButtonBlock>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithError: Story = {
  args: {
    errorMessage: 'Fix the highlighted settings before saving.',
  },
};
