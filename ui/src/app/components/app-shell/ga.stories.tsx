import type { Meta, StoryObj } from '@storybook/react-vite';
import { MemoryRouter } from 'react-router-dom';
import { GA } from './ga';

const meta = {
  title: 'App Shell/GA',
  component: GA,
  tags: ['autodocs'],
  decorators: [
    Story => (
      <MemoryRouter initialEntries={['/storybook/app-shell?tab=overview']}>
        <Story />
      </MemoryRouter>
    ),
  ],
  parameters: {
    docs: {
      description: {
        component:
          'Tracks route pageviews only for production, non-local hosts with a measurement id. Stories keep tracking disabled so Storybook never sends analytics events.',
      },
    },
  },
  argTypes: {
    env: {
      control: 'text',
      description: 'Runtime environment used by the analytics guard.',
    },
    hostname: {
      control: 'text',
      description: 'Browser hostname used to skip local development traffic.',
    },
    measurementId: {
      control: 'text',
      description: 'Google Analytics measurement id.',
    },
  },
  args: {
    env: 'development',
    hostname: 'localhost',
    measurementId: 'G-STORYBOOK',
  },
} satisfies Meta<typeof GA>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Development: Story = {};

export const Localhost: Story = {
  args: {
    env: 'production',
    hostname: 'localhost',
  },
};

export const MissingMeasurementId: Story = {
  args: {
    env: 'production',
    hostname: 'app.rapida.ai',
    measurementId: '',
  },
};
