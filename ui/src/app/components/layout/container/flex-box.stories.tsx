import type { Meta, StoryObj } from '@storybook/react-vite';
import { FlexBox } from './flex-box';

const meta = {
  title: 'Layout/Container/FlexBox',
  component: FlexBox,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Full-height linear page container used by public and auth routes.',
      },
    },
  },
  args: {
    showFooter: true,
    isFloatingHeader: false,
    children: (
      <section className="flex flex-1 items-center justify-center p-8">
        <p className="text-sm text-muted">Route content</p>
      </section>
    ),
  },
} satisfies Meta<typeof FlexBox>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithoutFooter: Story = {
  args: {
    showFooter: false,
  },
};

export const FloatingHeader: Story = {
  args: {
    isFloatingHeader: true,
  },
};
