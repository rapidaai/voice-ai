import { Metric } from '@rapidaai/react';
import { ExclamationTriangleIcon } from '@/app/components/Icon/exclamation-triangle';
import { PlayIcon } from '@/app/components/Icon/Play';
import { TickIcon } from '@/app/components/Icon/Tick';
import { Spinner } from '@/app/components/Loader/Spinner';
import { PlainWrapper } from '@/app/components/Wrapper/AlertWrapper';
import { cn } from '@/styles/media';
import React, { FC } from 'react';
import { FieldErrors } from 'react-hook-form';

export const ExecuteMessage: FC<{
  apiError?: string;
  loading?: boolean;
  formError?: FieldErrors;
  metrics: Array<Metric>;
  className?: string;
}> = ({ apiError, loading, formError, className, metrics }) => {
  if (loading) {
    return (
      <PlainWrapper className={cn(className, 'flex items-center')}>
        <Spinner className="w-5 h-5 text-blue-600 dark:text-blue-700" />
        <div className="text-sm">Executing your endpoint.</div>
      </PlainWrapper>
    );
  }
  if (apiError)
    return (
      <PlainWrapper className={className}>
        <ExclamationTriangleIcon className="w-5 h-5 text-red-600 dark:text-red-700" />
        <div className="text-sm text-red-600">{apiError}</div>
      </PlainWrapper>
    );

  if (formError && Object.entries(formError).length > 0)
    return (
      <PlainWrapper className={className}>
        <ExclamationTriangleIcon className="w-5 h-5 text-red-600 dark:text-red-700" />
        <ul className="text-sm text-red-600">
          {Object.entries(formError).map(([key, error]) => (
            <li key={key}>{error?.message?.toString()}</li>
          ))}
        </ul>
      </PlainWrapper>
    );
  if (metrics.length > 0) {
    return (
      <PlainWrapper className={className}>
        <TickIcon className="w-5 h-5 text-green-600 dark:text-green-700" />
        <div className="text-sm font-medium">Executed successfully.</div>
      </PlainWrapper>
    );
  }
  return (
    <PlainWrapper>
      <PlayIcon className="w-5 h-5 text-blue-600 dark:text-blue-700" />
      <div className="text-sm font-medium">
        Click on the button to execute endpoint.
      </div>
    </PlainWrapper>
  );
};
