import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  BlueNoticeBlock,
  GreenNoticeBlock,
  RedNoticeBlock,
  YellowNoticeBlock,
} from './notice-block';

const meta = {
  title: 'Layout/Container/NoticeBlock',
  component: BlueNoticeBlock,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Compact inline notice blocks for informational, success, warning, and error messages.',
      },
    },
  },
} satisfies Meta<typeof BlueNoticeBlock>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Info: Story = {
  args: {
    children: 'Documentation is available for this configuration.',
  },
};

export const Success: Story = {
  render: args => <GreenNoticeBlock {...args} />,
  args: {
    children: 'Credentials were saved.',
  },
};

export const Warning: Story = {
  render: args => <YellowNoticeBlock {...args} />,
  args: {
    children: 'Voice output is not enabled.',
  },
};

export const Error: Story = {
  render: args => <RedNoticeBlock {...args} />,
  args: {
    children: 'Fix the highlighted settings.',
  },
};
