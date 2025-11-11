import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';

import { EndpointIntegration } from '@/app/components/integration-document/endpoint-integration';
import { ModalProps } from '@/app/components/base/modal';
import { cn } from '@/styles/media';
import { HTMLAttributes } from 'react';

interface EndpointInstructionDialogProps
  extends ModalProps,
    HTMLAttributes<HTMLDivElement> {
  currentEndpoint?: Endpoint | null;
  currentEndpointProviderModel?: EndpointProviderModel | null;
}
export function EndpointInstructionDialog(
  props: EndpointInstructionDialogProps,
) {
  //   console.dir(props.currentEndpoint);
  const {
    currentEndpoint,
    currentEndpointProviderModel,
    className,
    ...mldAttr
  } = props;
  return (
    <RightSideModal
      className={cn(className)}
      {...mldAttr}
      title="Get started with endpoint"
    >
      <div className="relative overflow-auto h-[calc(100vh-100px)] px-4 pt-4 pb-10">
        {currentEndpoint && <EndpointIntegration endpoint={currentEndpoint} />}
      </div>
    </RightSideModal>
  );
}
