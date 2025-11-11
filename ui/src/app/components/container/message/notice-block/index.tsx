import { cn } from '@/styles/media';
import { HTMLAttributes } from 'react';

export const BlueNoticeBlock: React.FC<HTMLAttributes<HTMLDivElement>> = ({
  className,
  children,
}) => {
  return (
    <div
      className={cn(
        'px-4 py-2 border-[0.5px] border-l-4',
        'border-blue-600 bg-blue-500/10 text-sm',
        className,
      )}
    >
      {children}
    </div>
  );
};

export const GreenNoticeBlock: React.FC<HTMLAttributes<HTMLDivElement>> = ({
  className,
  children,
}) => {
  return (
    <div
      className={cn(
        'px-4 py-2 border-[0.5px] border-l-4',
        'border-green-600 bg-green-500/10 text-sm',
        className,
      )}
    >
      {children}
    </div>
  );
};

export const RedNoticeBlock: React.FC<HTMLAttributes<HTMLDivElement>> = ({
  className,
  children,
}) => {
  return (
    <div className="bg-white dark:bg-slate-950">
      <div
        className={cn(
          'px-4 py-2 border-[0.5px] border-l-4!',
          'text-sm border-red-600 bg-red-100 dark:bg-red-500/20',
          className,
        )}
      >
        {children}
      </div>
    </div>
  );
};

export const YellowNoticeBlock: React.FC<HTMLAttributes<HTMLDivElement>> = ({
  className,
  children,
}) => {
  return (
    <div
      className={cn(
        'px-4 py-2 border-[0.5px] border-l-4',
        'border-yellow-600 dark:border-yellow-600/70 bg-yellow-500/10 text-sm',
        className,
      )}
    >
      {children}
    </div>
  );
};
