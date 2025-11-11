import React from 'react';
import {
  CheckCircle,
  Clock,
  Archive,
  X,
  Activity,
  MinusCircle,
  Mail,
  Loader,
  Cable,
  LoaderCircle,
  CircleCheck,
} from 'lucide-react';

export const StatusIndicator = ({ state, size = 'medium' }) => {
  // Status configurations with SVG icons and appropriate colors
  const statusConfig = {
    INVITED: {
      bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
      textColor: 'text-yellow-700 dark:text-yellow-500',
      iconColor: 'text-yellow-500 dark:text-yellow-400',
      ringColor: 'ring-yellow-200 dark:ring-yellow-700',
      Icon: Mail, // Assuming you have a Mail icon imported
      display: 'Invited',
    },
    WAITLIST: {
      bgColor: 'bg-orange-100 dark:bg-orange-900/30',
      textColor: 'text-orange-700 dark:text-orange-500',
      iconColor: 'text-orange-500 dark:text-orange-400',
      ringColor: 'ring-orange-200 dark:ring-orange-700',
      Icon: Clock, // Reusing the Clock icon from IN_PROGRESS
      display: 'Waitlist',
    },

    ACTIVE: {
      bgColor: 'bg-green-100 dark:bg-green-900/30',
      textColor: 'text-green-700 dark:text-green-500',
      iconColor: 'text-green-500 dark:text-green-400',
      ringColor: 'ring-green-200 dark:ring-green-700',
      Icon: Activity,
      display: 'Active',
    },
    IN_PROGRESS: {
      bgColor: 'bg-blue-100 dark:bg-blue-900/30',
      textColor: 'text-blue-700 dark:text-blue-500',
      iconColor: 'text-blue-500 dark:text-blue-400 animate-spin3s',
      ringColor: 'ring-blue-200 dark:ring-blue-700',
      Icon: LoaderCircle,
      display: 'In progress',
    },
    SUCCESS: {
      bgColor: 'bg-emerald-100 dark:bg-emerald-900/30',
      textColor: 'text-emerald-700 dark:text-emerald-500',
      iconColor: 'text-emerald-500 dark:text-emerald-400',
      ringColor: 'ring-emerald-200/10 dark:ring-emerald-700/10',
      Icon: CircleCheck,
      display: 'Success',
    },
    COMPLETE: {
      bgColor: 'bg-purple-100 dark:bg-purple-900/30',
      textColor: 'text-purple-700 dark:text-purple-500',
      iconColor: 'text-purple-500 dark:text-purple-400',
      ringColor: 'ring-purple-200 dark:ring-purple-700',
      Icon: CircleCheck,
      display: 'Complete',
    },
    INACTIVE: {
      bgColor: 'bg-gray-100 dark:bg-gray-800/50',
      textColor: 'text-gray-700 dark:text-gray-500',
      iconColor: 'dark:text-gray-400',
      ringColor: 'ring-gray-200 dark:ring-gray-700',
      Icon: MinusCircle,
      display: 'Inactive',
    },
    ARCHIEVE: {
      bgColor: 'bg-amber-100 dark:bg-amber-900/30',
      textColor: 'text-amber-700 dark:text-amber-500',
      iconColor: 'text-amber-500 dark:text-amber-400',
      ringColor: 'ring-amber-200 dark:ring-amber-700',
      Icon: Archive,
      display: 'Archive',
    },
    QUEUED: {
      bgColor: 'bg-indigo-100 dark:bg-indigo-900/30',
      textColor: 'text-indigo-700 dark:text-indigo-500',
      iconColor: 'text-indigo-500 dark:text-indigo-400',
      ringColor: 'ring-indigo-200 dark:ring-indigo-700',
      Icon: Loader, // Assuming you have a Queue icon imported
      display: 'Queued',
    },
    CONNECTED: {
      bgColor: 'bg-teal-100 dark:bg-teal-900/30',
      textColor: 'text-teal-700 dark:text-teal-500',
      iconColor: 'text-teal-500 dark:text-teal-400',
      ringColor: 'ring-teal-200 dark:ring-teal-700',
      Icon: Cable, // Assuming you have a Link icon imported
      display: 'Connected',
    },
    FAILED: {
      bgColor: 'bg-red-100 dark:bg-red-900/30',
      textColor: 'text-red-700 dark:text-red-500',
      iconColor: 'text-red-500 dark:text-red-400',
      ringColor: 'ring-red-200 dark:ring-red-700',
      Icon: X,
      display: 'Failed',
    },
  };

  const config = statusConfig[state.toUpperCase()] || statusConfig['INACTIVE'];
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
      className={`inline-flex items-center rounded-[2px] ${config.bgColor} ${config.textColor} font-medium ${sizeClass.container} 5ing-[0.5px] ring-inset ${config.ringColor}`}
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
