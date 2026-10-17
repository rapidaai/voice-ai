import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { AnimatedLine } from './loaders/line-loader';
import { PageLoading } from './loading';
import { SectionLoader } from './loaders/section-loader';
import { Spinner } from './loaders/spinner';

const meta = {
  title: 'UI/Feedback/Loaders',
  component: PageLoading,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Reusable loading states for inline, section, and page-level waiting states.',
      },
    },
  },
} satisfies Meta<typeof PageLoading>;

export default meta;

type Story = StoryObj<typeof meta>;

export const LoadingStates: Story = {
  render: () => (
    <Stack gap={5}>
      <AnimatedLine animate="infinite" />
      <div className="flex items-center gap-6">
        <Spinner size="xs" />
        <Spinner size="sm" />
        <Spinner size="md" />
      </div>
      <SectionLoader />
      <PageLoading className="py-6" />
    </Stack>
  ),
};
