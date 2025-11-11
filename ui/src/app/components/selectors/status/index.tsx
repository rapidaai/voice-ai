import React from 'react';
import {
  Mail,
  Clock,
  Activity,
  CheckCircle,
  MinusCircle,
  Archive,
  Loader,
  Cable,
  X,
} from 'lucide-react';

// Status configurations with SVG icons and appropriate colors
const statusConfig = {
  INVITED: {
    bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
    textColor: 'text-yellow-700 dark:text-yellow-500',
    iconColor: 'text-yellow-500 dark:text-yellow-400',
    ringColor: 'ring-yellow-200 dark:ring-yellow-700',
    Icon: Mail,
    display: 'Invited',
  },
  WAITLIST: {
    bgColor: 'bg-orange-100 dark:bg-orange-900/30',
    textColor: 'text-orange-700 dark:text-orange-500',
    iconColor: 'text-orange-500 dark:text-orange-400',
    ringColor: 'ring-orange-200 dark:ring-orange-700',
    Icon: Clock,
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
    iconColor: 'text-blue-500 dark:text-blue-400',
    ringColor: 'ring-blue-200 dark:ring-blue-700',
    Icon: Clock,
    display: 'In progress',
  },
  SUCCESS: {
    bgColor: 'bg-emerald-100 dark:bg-emerald-900/30',
    textColor: 'text-emerald-700 dark:text-emerald-500',
    iconColor: 'text-emerald-500 dark:text-emerald-400',
    ringColor: 'ring-emerald-200/10 dark:ring-emerald-700/10',
    Icon: CheckCircle,
    display: 'Success',
  },
  COMPLETE: {
    bgColor: 'bg-purple-100 dark:bg-purple-900/30',
    textColor: 'text-purple-700 dark:text-purple-500',
    iconColor: 'text-purple-500 dark:text-purple-400',
    ringColor: 'ring-purple-200 dark:ring-purple-700',
    Icon: CheckCircle,
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
    Icon: Loader,
    display: 'Queued',
  },
  CONNECTED: {
    bgColor: 'bg-teal-100 dark:bg-teal-900/30',
    textColor: 'text-teal-700 dark:text-teal-500',
    iconColor: 'text-teal-500 dark:text-teal-400',
    ringColor: 'ring-teal-200 dark:ring-teal-700',
    Icon: Cable,
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

interface StatusSelectorProps {
  selectedStatus?: string;
  selectStatus: (status: string) => void;
}

const StatusSelector: React.FC<StatusSelectorProps> = ({
  selectedStatus,
  selectStatus,
}) => {
  return (
    <div className="flex flex-col space-y-2">
      {Object.entries(statusConfig).map(([value, config]) => (
        <button
          key={value}
          onClick={() => selectStatus(value)}
          className={`w-full flex items-center gap-3 p-3 border transition-all ${
            selectedStatus === value
              ? `${config.bgColor} ${config.ringColor} ${config.textColor} border-l-4`
              : 'bg-light-background dark:bg-gray-950'
          }`}
        >
          <config.Icon className={`w-4 h-4 ${config.iconColor}`} />
          <span className="text-sm font-medium">{config.display}</span>
          {selectedStatus === value && (
            <div
              className={`ml-auto w-2 h-2 ${config.iconColor.replace('text', 'bg')} rounded-[2px]`}
            />
          )}
        </button>
      ))}
    </div>
  );
};

export default StatusSelector;
