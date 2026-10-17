import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  AssistantWebwidgetDeploymentDialog,
  ConfigureAssistantTemplateDialog,
  ConfigureAssistantToolDialog,
  DebuggerDeploymentSuccessDialog,
} from '.';

const meta = {
  title: 'Dialogs/Assistant',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Assistant dialogs for template selection, tool configuration, and deployment guidance.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const TemplateSelection: Story = {
  render: () => (
    <ConfigureAssistantTemplateDialog
      modalOpen
      setModalOpen={() => undefined}
      onSelectTemplate={() => undefined}
    />
  ),
};

export const ConfigureTool: Story = {
  render: () => (
    <ConfigureAssistantToolDialog
      modalOpen
      setModalOpen={() => undefined}
      initialData={null}
      onChange={() => undefined}
    />
  ),
};

export const WebWidgetDeployment: Story = {
  render: () => (
    <AssistantWebwidgetDeploymentDialog
      modalOpen
      setModalOpen={() => undefined}
      assistantId="assistant-123"
    />
  ),
};

export const DebuggerSuccess: Story = {
  render: () => (
    <DebuggerDeploymentSuccessDialog
      modalOpen
      setModalOpen={() => undefined}
      assistantId="assistant-123"
    />
  ),
};
