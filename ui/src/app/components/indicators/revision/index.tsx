import React, { FC } from 'react';
import { Check, Clock, MinusCircle } from 'lucide-react';

export const RevisionIndicator: FC<{
  status: 'DEPLOYED' | 'NOT_DEPLOYED' | 'DEPLOYING';
  size?: 'small' | 'medium' | 'large';
}> = ({ status, size = 'medium' }) => {
  const statusConfig = {
    DEPLOYED: {
      bgColor: 'bg-green-100 dark:bg-green-900/30',
      textColor: 'text-green-700 dark:text-green-500',
      iconColor: 'text-green-500 dark:text-green-400',
      ringColor: 'ring-green-200 dark:ring-green-700',
      Icon: Check,
      display: 'In use',
    },
    NOT_DEPLOYED: {
      bgColor: 'bg-gray-100 dark:bg-gray-800/50',
      textColor: 'text-gray-700 dark:text-gray-500',
      iconColor: 'dark:text-gray-400',
      ringColor: 'ring-gray-200 dark:ring-gray-700',
      Icon: MinusCircle,
      display: 'Not in use',
    },
    DEPLOYING: {
      bgColor: 'bg-blue-100 dark:bg-blue-900/30',
      textColor: 'text-blue-700 dark:text-blue-500',
      iconColor: 'text-blue-500 dark:text-blue-400',
      ringColor: 'ring-blue-200 dark:ring-blue-700',
      Icon: Clock,
      display: 'Deploying',
    },
  };

  const config = statusConfig[status] || statusConfig['NOT_DEPLOYED'];
  const { Icon } = config;

  // Size variants
  const sizeClasses = {
    small: {
      container: 'text-xs px-2 py-0.5 gap-1',
      icon: 12,
    },
    medium: {
      container: 'text-sm px-2.5 py-1 gap-1.5',
      icon: 16,
    },
    large: {
      container: 'text-base px-3 py-1.5 gap-2',
      icon: 18,
    },
  };

  const sizeClass = sizeClasses[size] || sizeClasses.medium;

  return (
    <span
      className={`shrink-0 inline-flex items-center rounded-[2px] ${config.bgColor} ${config.textColor} font-medium ${sizeClass.container} ring-[0.5px] ring-inset ${config.ringColor}`}
    >
      <Icon
        className={`${config.iconColor}`}
        size={sizeClass.icon}
        strokeWidth={1.5}
      />
      {config.display}
    </span>
  );
};
