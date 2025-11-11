import React, { useEffect } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { cn } from '@/styles/media';
/**
 *
 */
export interface ModalProps {
  /**
   * Modal control
   */
  modalOpen: boolean;

  /**
   *
   * @param boolean
   * @returns
   */
  setModalOpen: (boolean) => void;
}

interface GenericModalProps extends ModalProps {
  children: any;
  className?: string;
  transitionBackdropClass: string;
  transitionDialogClass: string;

  whenEnter: string;
  whenEnterStart: string;
  whenEnterEnd: string;
  whenLeave: string;
  whenLeaveStart: string;
  whenLeaveEnd: string;
}

export function GenericModal(props: GenericModalProps) {
  // auto close modal feature
  useEffect(() => {
    const keyHandler = ({ keyCode }) => {
      if (!props.modalOpen || keyCode !== 27) return;
      props.setModalOpen(false);
    };
    document.addEventListener('keydown', keyHandler);
    return () => document.removeEventListener('keydown', keyHandler);
  });

  return (
    <AnimatePresence>
      {props.modalOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={() => props.setModalOpen(false)}
          className={cn(
            'bg-slate-900/20 dark:bg-slate-400/5 backdrop-blur-xs p-8 fixed inset-0 z-50 grid place-items-center overflow-y-scroll cursor-pointer',
            props.className,
          )}
        >
          <motion.div
            onClick={e => e.stopPropagation()}
            className={props.transitionDialogClass}
          >
            {props.children}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

GenericModal.defaultProps = {
  transitionBackdropClass:
    'fixed inset-0 backdrop-blur-xs dark:bg-gray-500/30 bg-gray-800/10 z-50 transition-opacity w-100 h-100 flex items-center',
  transitionDialogClass:
    'fixed inset-0 z-50 overflow-hidden flex items-center justify-center px-4 sm:px-6',
  whenEnter: 'transition ease-in-out duration-200',
  whenEnterStart: 'opacity-0 trangray-y-4',
  whenEnterEnd: 'opacity-100 trangray-y-0',
  whenLeave: 'transition ease-in-out duration-200',
  whenLeaveStart: 'opacity-100 trangray-y-0',
  whenLeaveEnd: 'opacity-0 trangray-y-4',
};
