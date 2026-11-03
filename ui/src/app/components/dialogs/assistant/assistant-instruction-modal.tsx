import { memo } from 'react';
import type { HTMLAttributes } from 'react';
import type { ModalProps } from '@/app/components/ui/primitives';
import { RightSideModal } from '@/app/components/dialogs/shared';
import { WidgetIntegration } from '@/app/components/domain/integration-document/widget-integration';

interface AssistantInstructionDialogProps
  extends ModalProps,
    HTMLAttributes<HTMLDivElement> {
  assistantId: string;
}

function AssistantWebwidgetDeploymentDialogComponent({
  assistantId,
  modalOpen,
  setModalOpen,
}: AssistantInstructionDialogProps) {
  return (
    <RightSideModal
      modalOpen={modalOpen}
      setModalOpen={setModalOpen}
      className="w-[580px] max-w-[calc(100vw-2rem)]"
      label="Web Widget Deployment"
      title="Deployment completed"
    >
      <div className="relative flex min-h-0 flex-1 flex-col overflow-auto">
        <WidgetIntegration assistantId={assistantId} />
      </div>
    </RightSideModal>
  );
}

export const AssistantWebwidgetDeploymentDialog = memo(
  AssistantWebwidgetDeploymentDialogComponent,
);
