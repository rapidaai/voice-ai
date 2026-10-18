import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { CarbonIconIndicator } from './icon-indicator';
import { RecordStatusIndicator } from './record-status-indicator';
import { CarbonShapeIndicator } from './shape-indicator';
import { CarbonStatusIndicator } from './status-indicator';

const meta = {
  title: 'UI/Feedback/Indicators',
  component: CarbonStatusIndicator,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Carbon status indicator wrappers used for record, workflow, and call state display.',
      },
    },
  },
  args: {
    state: 'SUCCESS',
  },
} satisfies Meta<typeof CarbonStatusIndicator>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Statuses: Story = {
  render: () => (
    <Stack gap={4}>
      <CarbonStatusIndicator state="SUCCESS" />
      <CarbonStatusIndicator state="IN_PROGRESS" />
      <CarbonStatusIndicator state="RINGING" />
      <CarbonStatusIndicator state="FAILED" />
      <CarbonStatusIndicator state="UNKNOWN_STATUS" />
    </Stack>
  ),
};

export const RecordStatuses: Story = {
  render: () => (
    <Stack gap={4}>
      <RecordStatusIndicator state="ACTIVE" />
      <RecordStatusIndicator state="INACTIVE" />
      <RecordStatusIndicator state="DRAFT" />
    </Stack>
  ),
};

export const ShapeAndIcon: Story = {
  render: () => (
    <Stack gap={4}>
      <CarbonShapeIndicator state="RECORD_ACTIVE" />
      <CarbonShapeIndicator state="RECORD_FAILED" />
      <CarbonIconIndicator state="RECORD_CONNECTED" />
      <CarbonIconIndicator state="RECORD_FAILED" />
    </Stack>
  ),
};
