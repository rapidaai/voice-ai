import type { Meta, StoryObj } from '@storybook/react-vite';
import { DocNoticeBlock } from './doc-notice-block';

const meta = {
  title: 'Layout/Container/DocNoticeBlock',
  component: DocNoticeBlock,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Documentation notice with an inline Carbon link resolved from the active theme.',
      },
    },
  },
  args: {
    docPath: '/assistants/create-new-version',
    children: 'Use documentation for required assistant configuration fields.',
  },
} satisfies Meta<typeof DocNoticeBlock>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const ExplicitUrl: Story = {
  args: {
    docUrl: 'https://docs.example.com/assistants/create-new-version',
    linkText: 'Open docs',
  },
};
