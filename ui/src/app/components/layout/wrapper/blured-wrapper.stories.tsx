import type { Meta, StoryObj } from '@storybook/react-vite';
import { Button, Tag } from '@carbon/react';
import { BluredWrapper } from './blured-wrapper';

const meta = {
  title: 'Layout/Wrapper/BluredWrapper',
  component: BluredWrapper,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Header-style wrapper used for sticky table and detail page controls.',
      },
    },
  },
} satisfies Meta<typeof BluredWrapper>;

export default meta;

type Story = StoryObj<typeof meta>;

export const HeaderControls: Story = {
  args: {
    className: 'p-3',
    children: (
      <>
        <div className="flex items-center gap-2">
          <strong>Knowledge documents</strong>
          <Tag size="sm" type="cyan">
            24
          </Tag>
        </div>
        <Button kind="secondary" size="sm">
          Upload
        </Button>
      </>
    ),
  },
};
