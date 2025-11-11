import { RedNoticeBlock } from '@/app/components/container/message/notice-block';
import { AlertTriangle } from '@/app/components/Icon/alert-triangle';
import { FC, HTMLAttributes } from 'react';

export const PageActionButtonBlock: FC<
  {
    errorMessage?: string;
  } & HTMLAttributes<HTMLDivElement>
> = ({ errorMessage, children }) => {
  return (
    <div className="absolute bottom-0 left-0 right-0 ">
      {errorMessage && (
        <RedNoticeBlock className="flex items-center space-x-2 ">
          <AlertTriangle className="w-4 h-4" />
          <span>{errorMessage}</span>
        </RedNoticeBlock>
      )}
      <div className="flex items-center justify-end border-t dark:bg-gray-900 bg-white">
        <div className="flex space-x-2 py-2 px-4">{children}</div>
      </div>
    </div>
  );
};
