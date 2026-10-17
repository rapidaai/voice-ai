import type { Meta, StoryObj } from '@storybook/react-vite';
import { Helmet } from './helmet';

const meta = {
  title: 'App Shell/Helmet',
  component: Helmet,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Sets document title and route metadata from the active application theme.',
      },
    },
  },
  argTypes: {
    title: {
      control: 'text',
      description:
        'Page title. Empty or whitespace-only values fall back to the active brand name.',
    },
    meta: {
      control: 'object',
      description:
        'Additional document meta entries rendered as name/content tags.',
    },
  },
  args: {
    title: 'Dashboard',
    meta: [{ name: 'description', content: 'Tenant dashboard' }],
  },
} satisfies Meta<typeof Helmet>;

export default meta;

type Story = StoryObj<typeof meta>;

export const PageTitle: Story = {};

export const BrandOnly: Story = {
  args: {
    title: '',
    meta: [],
  },
};

export const WithMetaTags: Story = {
  args: {
    title: 'Assistant analytics',
    meta: [
      { name: 'description', content: 'Assistant analytics dashboard' },
      { name: 'og:title', content: 'Assistant analytics' },
    ],
  },
};
