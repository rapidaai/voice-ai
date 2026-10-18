import type { Meta, StoryObj } from '@storybook/react-vite';
import { ErrorContainer } from './error-container';

const meta = {
  title: 'UI/Feedback/ErrorContainer',
  component: ErrorContainer,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Page-level error state with a single recovery action and Carbon button styling.',
      },
    },
  },
  args: {
    code: '404',
    title: 'Page not found',
    description: 'The page may have moved or you may not have access.',
    actionLabel: 'Go back',
    onAction: () => undefined,
  },
} satisfies Meta<typeof ErrorContainer>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const ServerError: Story = {
  args: {
    code: '500',
    title: 'Something went wrong',
    description: 'The request could not be completed. Try again in a moment.',
    actionLabel: 'Retry',
  },
};
