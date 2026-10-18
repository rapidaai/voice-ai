import type { Meta, StoryObj } from '@storybook/react-vite';
import { Button } from '@carbon/react';
import { useState } from 'react';
import { RightSideModal } from './right-side-modal';
import { ModalBody } from './modal-body';
import { OverviewRow } from './overview-row';

const meta = {
  title: 'Dialogs/Shared/RightSideModal',
  component: RightSideModal,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Drawer-style modal for inspecting details without leaving the current page.',
      },
    },
  },
} satisfies Meta<typeof RightSideModal>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Drawer: Story = {
  args: {
    modalOpen: true,
    setModalOpen: () => undefined,
    children: null,
  },
  render: function Render() {
    const [modalOpen, setModalOpen] = useState(true);

    return (
      <div className="min-h-96 bg-background p-4">
        <Button size="sm" onClick={() => setModalOpen(true)}>
          Open drawer
        </Button>
        <RightSideModal
          modalOpen={modalOpen}
          setModalOpen={setModalOpen}
          title="Trace details"
          label="Endpoint"
          className="w-[28rem]"
        >
          <ModalBody>
            <OverviewRow label="Status">Completed</OverviewRow>
            <OverviewRow label="Latency">412 ms</OverviewRow>
          </ModalBody>
        </RightSideModal>
      </div>
    );
  },
};
