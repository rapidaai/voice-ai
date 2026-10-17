import type { Meta, StoryObj } from '@storybook/react-vite';
import { SectionDivider } from './section-divider';

const meta = {
  title: 'Layout/Blocks/SectionDivider',
  component: SectionDivider,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component: 'Small section label with a horizontal divider line.',
      },
    },
  },
  args: {
    label: 'Provider',
  },
} satisfies Meta<typeof SectionDivider>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
