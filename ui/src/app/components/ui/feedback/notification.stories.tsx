import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import {
  ActionNotification,
  LinkNotification,
  Notification,
} from './notification';
import { ErrorMessage } from './error-message';
import { ToastNotification } from './toast';

const meta = {
  title: 'UI/Feedback/Notifications',
  component: Notification,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Carbon notification wrappers for inline, actionable, link-style, error, and toast feedback.',
      },
    },
  },
  args: {
    kind: 'info',
    title: 'Information',
  },
} satisfies Meta<typeof Notification>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Inline: Story = {
  render: () => (
    <Stack gap={4}>
      <Notification kind="info" title="Information" subtitle="Details saved." />
      <Notification
        kind="success"
        title="Success"
        subtitle="Deployment is ready."
      />
      <ErrorMessage message="Unable to load provider credentials." />
    </Stack>
  ),
};

export const Actions: Story = {
  render: () => (
    <Stack gap={4}>
      <ActionNotification
        kind="warning"
        title="Review required"
        subtitle="A required setting is missing."
        actionButtonLabel="Review"
        onActionButtonClick={() => undefined}
      />
      <LinkNotification
        kind="info"
        title="Documentation"
        subtitle="Review setup guidance before continuing."
        linkText="Open docs"
        onLinkClick={() => undefined}
      />
      <ToastNotification
        kind="success"
        title="Saved"
        subtitle="Your changes are available."
      />
    </Stack>
  ),
};
