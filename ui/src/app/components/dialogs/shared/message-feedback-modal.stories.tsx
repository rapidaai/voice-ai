import type { Meta, StoryObj } from '@storybook/react-vite';
import { MessageFeedbackDialog } from './message-feedback-modal';

const meta = {
  title: 'Dialogs/Shared/MessageFeedbackDialog',
  component: MessageFeedbackDialog,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Feedback dialog for collecting short free-form response feedback.',
      },
    },
  },
  args: {
    modalOpen: true,
    setModalOpen: () => undefined,
    onSubmitFeedback: () => undefined,
  },
} satisfies Meta<typeof MessageFeedbackDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Open: Story = {};
