import type { Meta, StoryObj } from '@storybook/react-vite';
import { Button } from '@carbon/react';
import { CompactPageHeader } from './compact-page-header';
import { PageTitleBlock } from './blocks/page-title-block';

const meta = {
  title: 'Layout/CompactPageHeader',
  component: CompactPageHeader,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Compact semantic header for dense internal work areas and page-specific actions.',
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
} satisfies Meta<typeof CompactPageHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
