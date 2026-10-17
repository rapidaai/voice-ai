import type { Meta, StoryObj } from '@storybook/react-vite';
import { Button } from '@carbon/react';
import { PageHeaderBlock } from './page-header-block';
import { PageTitleBlock } from './page-title-block';

const meta = {
  title: 'Layout/Blocks/PageHeaderBlock',
  component: PageHeaderBlock,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Page header row for a title area and right-aligned page actions.',
      },
    },
  },
  args: {
    children: (
      <>
        <PageTitleBlock>Assistants</PageTitleBlock>
        <Button kind="primary" size="sm">
          Create
        </Button>
      </>
    ),
  },
} satisfies Meta<typeof PageHeaderBlock>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
