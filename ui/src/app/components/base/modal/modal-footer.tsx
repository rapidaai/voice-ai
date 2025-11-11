import { RedNoticeBlock } from '@/app/components/container/message/notice-block';
import { AlertTriangle } from '@/app/components/Icon/alert-triangle';
import { cn } from '@/styles/media';
import { FC, HTMLAttributes } from 'react';

interface ModalFooterProps extends HTMLAttributes<HTMLDivElement> {
  errorMessage?: string;
}

export const ModalFooter: FC<ModalFooterProps> = props => {
  return (
    <div className="flex flex-col">
      {props.errorMessage && (
        <RedNoticeBlock className="flex items-center space-x-2">
          <AlertTriangle className="w-4 h-4" />
          <span>{props.errorMessage}</span>
        </RedNoticeBlock>
      )}
      <div
        className={cn(
          'border-t dark:border-gray-800 flex justify-end px-4 py-2 space-x-2',
          props.className,
        )}
      >
        {props.children}
      </div>
    </div>
  );
};
