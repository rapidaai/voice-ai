import type { Meta, StoryObj } from '@storybook/react-vite';
import { CloudUpload, Settings, TaskComplete } from '@carbon/icons-react';
import { HowItWorks, HowItWorksDialog } from './how-it-works-modal';

const steps = [
  {
    title: 'Connect source',
    icon: <CloudUpload />,
    description: 'Choose the data or provider that should be configured.',
  },
  {
    title: 'Configure options',
    icon: <Settings />,
    description: 'Review the required settings before the deployment starts.',
  },
  {
    title: 'Finish setup',
    icon: <TaskComplete />,
    description: 'Save the configuration and continue with testing.',
  },
];

const meta = {
  title: 'Dialogs/Shared/HowItWorks',
  component: HowItWorksDialog,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Reusable modal content for short step-by-step setup explanations.',
      },
    },
  },
} satisfies Meta<typeof HowItWorksDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Content: Story = {
  args: {
    modalOpen: false,
    setModalOpen: () => undefined,
    steps,
  },
  render: () => (
    <div className="max-w-4xl bg-layer text-foreground">
      <HowItWorks steps={steps} />
    </div>
  ),
};

export const Dialog: Story = {
  args: {
    modalOpen: true,
    setModalOpen: () => undefined,
    title: 'How deployment works',
    steps,
  },
};
