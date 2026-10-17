import type { Meta, StoryObj } from '@storybook/react-vite';
import { Button, Checkbox } from '@carbon/react';
import SingleRowWrapper from './single-row-wrapper';

const meta = {
  title: 'Layout/Wrapper/SingleRowWrapper',
  component: SingleRowWrapper,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Compact row wrapper for file rows and small inline action groups.',
      },
    },
  },
} satisfies Meta<typeof SingleRowWrapper>;

export default meta;

type Story = StoryObj<typeof meta>;

export const FileRow: Story = {
  args: {
    className: 'max-w-xl',
    children: (
      <>
        <Checkbox id="manual-file-row" labelText="manual-upload.csv" />
        <Button kind="ghost" size="sm">
          Remove
        </Button>
      </>
    ),
  },
};
