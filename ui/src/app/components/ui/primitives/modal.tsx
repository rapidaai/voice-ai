import { useEffect } from 'react';
import type { MouseEvent, ReactNode } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import {
  ComposedModal,
  ModalHeader as CarbonModalHeader,
  ModalBody as CarbonModalBody,
  ModalFooter as CarbonModalFooter,
} from '@carbon/react';
import { cn } from '@/utils';

type ModalSize = 'xs' | 'sm' | 'md' | 'lg';

export interface ModalProps {
  modalOpen: boolean;
  setModalOpen: (modalOpen: boolean) => void;
}

interface GenericModalProps extends ModalProps {
  children: ReactNode;
  className?: string;
  transitionBackdropClass?: string;
  transitionDialogClass?: string;
  whenEnter?: string;
  whenEnterStart?: string;
  whenEnterEnd?: string;
  whenLeave?: string;
  whenLeaveStart?: string;
  whenLeaveEnd?: string;
}

export interface CarbonModalProps {
  open: boolean;
  onClose: (e?: MouseEvent) => void;
  className?: string;
  containerClassName?: string;
  size?: ModalSize;
  danger?: boolean;
  preventCloseOnClickOutside?: boolean;
  selectorPrimaryFocus?: string;
  'aria-label'?: string;
  children: ReactNode;
}

export interface CarbonModalHeaderProps {
  className?: string;
  title?: string;
  titleClassName?: string;
  label?: string;
  labelClassName?: string;
  children?: ReactNode;
  onClose?: () => void;
}

export interface CarbonModalBodyProps {
  className?: string;
  children: ReactNode;
  hasForm?: boolean;
  hasScrollingContent?: boolean;
}

export interface CarbonModalFooterProps {
  className?: string;
  children: ReactNode;
  danger?: boolean;
}

export function GenericModal(props: GenericModalProps) {
  useEffect(() => {
    const keyHandler = ({ keyCode }: KeyboardEvent) => {
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
            'bg-slate-900/20 dark:bg-slate-400/5 backdrop-blur-xs p-8 fixed inset-0 z-50 grid place-items-center overflow-y-scroll',
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

export function Modal({
  open,
  onClose,
  className,
  containerClassName,
  size = 'md',
  danger = false,
  preventCloseOnClickOutside = false,
  selectorPrimaryFocus,
  children,
  ...rest
}: CarbonModalProps) {
  return (
    <ComposedModal
      open={open}
      onClose={onClose}
      className={cn(className)}
      containerClassName={cn(containerClassName)}
      size={size}
      danger={danger}
      preventCloseOnClickOutside={preventCloseOnClickOutside}
      selectorPrimaryFocus={selectorPrimaryFocus}
      {...rest}
    >
      {children}
    </ComposedModal>
  );
}

export function ModalHeader({
  className,
  title,
  titleClassName,
  label,
  labelClassName,
  children,
  onClose,
}: CarbonModalHeaderProps) {
  return (
    <CarbonModalHeader
      className={cn(className)}
      title={title}
      titleClassName={cn(titleClassName)}
      label={label}
      labelClassName={cn(labelClassName)}
      buttonOnClick={onClose}
    >
      {children}
    </CarbonModalHeader>
  );
}

export function ModalBody({
  className,
  children,
  hasForm = false,
  hasScrollingContent = false,
}: CarbonModalBodyProps) {
  return (
    <CarbonModalBody
      className={cn(className)}
      hasForm={hasForm}
      hasScrollingContent={hasScrollingContent}
    >
      {children}
    </CarbonModalBody>
  );
}

export function ModalFooter({
  className,
  children,
  danger = false,
}: CarbonModalFooterProps) {
  return (
    <CarbonModalFooter className={cn(className)} danger={danger}>
      {children}
    </CarbonModalFooter>
  );
}
