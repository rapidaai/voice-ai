import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  DeploymentRow,
  DeploymentSectionHeader,
} from './deployment-modal-primitives';
import { ModalBody } from './modal-body';
import { OverviewRow } from './overview-row';

const meta = {
  title: 'Dialogs/Shared/DialogPrimitives',
  component: ModalBody,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Small reusable row and body primitives used by shared dialog layouts.',
      },
    },
  },
} satisfies Meta<typeof ModalBody>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Rows: Story = {
  render: () => (
    <div className="max-w-xl border border-border-subtle bg-surface text-foreground">
      <DeploymentSectionHeader label="Deployment" />
      <DeploymentRow label="Endpoint">production-api</DeploymentRow>
      <DeploymentRow label="Region">us-east</DeploymentRow>
      <DeploymentSectionHeader label="Overview" />
      <OverviewRow label="Status">Active</OverviewRow>
    </div>
  ),
};
