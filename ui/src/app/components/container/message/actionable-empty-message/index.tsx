import { IBlueBorderPlusButton } from '@/app/components/Form/Button';
import { FC } from 'react';

export const ActionableEmptyMessage: FC<{
  title: string;
  subtitle: string;
  action?: string;
  actionComponent?: any;
  onActionClick?: () => void;
}> = ({ title, subtitle, action, actionComponent, onActionClick }) => {
  return (
    <div className="px-4 py-6 flex flex-col justify-center items-center">
      <div className="font-semibold">{title}</div>
      <div>{subtitle}</div>
      {actionComponent && actionComponent}
      {action && (
        <IBlueBorderPlusButton
          onClick={onActionClick}
          className="mt-3 bg-white"
        >
          {action}
        </IBlueBorderPlusButton>
      )}
    </div>
  );
};
