import type { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { RightSideModal } from '@/app/components/dialogs/shared';
import { EndpointIntegration } from '@/app/components/domain/integration-document/endpoint-integration';
import type { ModalProps } from '@/app/components/ui/primitives';
import { cn } from '@/utils';
import type { HTMLAttributes } from 'react';

interface EndpointInstructionDialogProps
  extends ModalProps,
    HTMLAttributes<HTMLDivElement> {
  currentEndpoint?: Endpoint | null;
  currentEndpointProviderModel?: EndpointProviderModel | null;
}

export function EndpointInstructionDialog(
  props: EndpointInstructionDialogProps,
) {
  const {
    currentEndpoint,
    currentEndpointProviderModel,
    className,
    ...mldAttr
  } = props;

  return (
    <RightSideModal
      className={cn('w-[580px] max-w-[calc(100vw-2rem)]', className)}
      label="Integration"
      title={currentEndpoint?.getName() || 'Get started'}
      {...mldAttr}
    >
      <div className="relative flex min-h-0 flex-1 flex-col">
        {currentEndpoint && <EndpointIntegration endpoint={currentEndpoint} />}
      </div>
    </RightSideModal>
  );
}
