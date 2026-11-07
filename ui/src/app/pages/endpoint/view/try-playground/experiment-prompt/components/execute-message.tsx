import { Metric } from '@rapidaai/react';
import { cn } from '@/utils';
import { InlineLoading, InlineNotification } from '@carbon/react';
import { FC } from 'react';
import { FieldErrors } from 'react-hook-form';

export const ExecuteMessage: FC<{
  apiError?: string;
  loading?: boolean;
  formError?: FieldErrors;
  metrics: Array<Metric>;
  className?: string;
}> = ({ apiError, loading, formError, className, metrics }) => {
  const messageClassName = cn('!m-0 !max-w-full', className);

  if (loading) {
    return (
      <div
        className={cn(
          'flex min-h-12 items-center border-b border-border-subtle bg-layer px-4',
          className,
        )}
      >
        <InlineLoading
          status="active"
          iconDescription="Executing endpoint"
          description="Executing your endpoint."
        />
      </div>
    );
  }

  if (apiError) {
    return (
      <InlineNotification
        className={messageClassName}
        kind="error"
        lowContrast
        hideCloseButton
        role="alert"
        title="Endpoint execution failed"
        subtitle={apiError}
      />
    );
  }

  const formErrors = Object.values(formError ?? {})
    .map(error => error?.message?.toString())
    .filter((message): message is string => Boolean(message));

  if (formErrors.length > 0) {
    return (
      <InlineNotification
        className={messageClassName}
        kind="warning"
        lowContrast
        hideCloseButton
        title="Check the endpoint arguments"
        subtitle={formErrors.join(' ')}
      />
    );
  }

  if (metrics.length > 0) {
    return (
      <InlineNotification
        className={messageClassName}
        kind="success"
        lowContrast
        hideCloseButton
        title="Endpoint executed"
        subtitle="The response and execution details are ready below."
      />
    );
  }

  return (
    <InlineNotification
      className={messageClassName}
      kind="info"
      lowContrast
      hideCloseButton
      title="Ready to run"
      subtitle="Run the endpoint to view its output, metadata, and metrics."
    />
  );
};
