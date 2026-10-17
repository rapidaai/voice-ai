import type { Meta, StoryObj } from '@storybook/react-vite';
import { DescriptiveHeading } from './descriptive-heading';

const meta = {
  title: 'Layout/Heading/DescriptiveHeading',
  component: DescriptiveHeading,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Page-level heading with optional parenthetical context and supporting description text.',
      },
    },
  },
  args: {
    heading: 'Organization Profile',
    subheading: 'Update organization identity and operational contact details.',
  },
  argTypes: {
    level: {
      control: 'inline-radio',
      options: [1, 2, 3, 4, 5, 6],
    },
  },
} satisfies Meta<typeof DescriptiveHeading>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithInfo: Story = {
  args: {
    heading: 'Archive Organization',
    info: 'Danger zone',
    subheading:
      'Archive the organization only after all active projects have been reviewed.',
  },
};

export const SectionHeading: Story = {
  args: {
    heading: 'Security',
    level: 2,
    subheading: 'Manage authentication and access controls.',
  },
};
