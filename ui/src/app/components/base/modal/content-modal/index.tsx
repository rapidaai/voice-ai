import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { Spinner } from '@/app/components/Loader/Spinner';
import React, { ReactElement } from 'react';
import { cn } from '@/styles/media';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { MoveRight } from 'lucide-react';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';

/**
 *
 */
interface CenterModalProps
  extends React.FormHTMLAttributes<HTMLFormElement>,
    ModalProps {
  title?: string;
  action?: any;
  actionWrapper: (any) => ReactElement;
  children: ReactElement;
  loading: boolean;
}

/**
 *
 * @param props
 * @returns
 */
export function CenterModal(props: CenterModalProps) {
  // close if the esc key is pressed

  return (
    <GenericModal modalOpen={props.modalOpen} setModalOpen={props.setModalOpen}>
      <ModalFitHeightBlock>
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
        >
          <ModalTitleBlock>{props.title}</ModalTitleBlock>
        </ModalHeader>
        <ModalBody className={cn('relative', props.className)}>
          <form onSubmit={props.onSubmit} className={cn('relative')}>
            <div className="max-h-[500px] overflow-auto">
              {props.loading && (
                <div className="absolute w-full h-full backdrop-blur-xs bg-white/30 dark:bg-white/5 z-50 justify-center items-center flex">
                  <Spinner size="md"></Spinner>
                </div>
              )}
              <div className={cn('px-4 py-6')}>{props.children}</div>
            </div>
          </form>
        </ModalBody>
        <ModalFooter>
          <ICancelButton
            className="px-4 rounded-[2px]"
            onClick={() => {
              props.setModalOpen(false);
            }}
          >
            Cancel
          </ICancelButton>
          {props.actionWrapper(props.action)}
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
}

CenterModal.defaultProps = {
  loading: false,
  className: 'flex flex-col w-full md:max-w-xl relative shadow-xs z-50',
  actionWrapper: (action: any, loading?: boolean) => {
    return (
      <IBlueBGButton
        className="px-4 rounded-[2px]"
        type="submit"
        isLoading={loading}
      >
        {action}
        <MoveRight className="ml-2" strokeWidth={1.5} />
      </IBlueBGButton>
    );
  },
};
