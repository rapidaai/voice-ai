import type { Meta, StoryObj } from '@storybook/react-vite';
import { GeneralFooter } from './general-footer';

const meta = {
  title: 'Layout/Footer/GeneralFooter',
  component: GeneralFooter,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Footer shell that renders theme-provided legal, documentation, source, and support links.',
      },
    },
  },
} satisfies Meta<typeof GeneralFooter>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: args => (
    <div className="flex min-h-32 items-end bg-background">
      <GeneralFooter {...args} className="w-full" />
    </div>
  ),
};
