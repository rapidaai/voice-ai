import { CheckCircle, XCircle, AlertTriangle } from 'lucide-react'; // Import necessary icons

// Status configurations with SVG icons and appropriate colors
const statusConfig = {
  SUCCESS: {
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    textColor: 'text-green-700 dark:text-green-500',
    iconColor: 'text-green-500 dark:text-green-400',
    ringColor: 'ring-green-200/10 dark:ring-green-700/10',
    Icon: CheckCircle,
    display: 'Success',
  },
  CLIENT_ERROR: {
    bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
    textColor: 'text-yellow-700 dark:text-yellow-500',
    iconColor: 'text-yellow-500 dark:text-yellow-400',
    ringColor: 'ring-yellow-200/10 dark:ring-yellow-700/10',
    Icon: AlertTriangle,
    display: 'Client Error',
  },
  SERVER_ERROR: {
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    textColor: 'text-red-700 dark:text-red-500',
    iconColor: 'text-red-500 dark:text-red-400',
    ringColor: 'ring-red-200/10 dark:ring-red-700/10',
    Icon: XCircle,
    display: 'Server Error',
  },
};

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

export const HttpStatusSpanIndicator = ({
  status,
  size = 'medium',
}: {
  status: number;
  size?: 'small' | 'medium' | 'large';
}) => {
  const getStatusType = (status: number) => {
    if (status >= 200 && status < 300) return 'SUCCESS';
    if (status >= 400 && status < 500) return 'CLIENT_ERROR';
    if (status >= 500) return 'SERVER_ERROR';
    return 'CLIENT_ERROR'; // Default to CLIENT_ERROR for unknown status codes
  };

  const statusType = getStatusType(status);
  const config = statusConfig[statusType];
  const { Icon } = config;
  const sizeClass = sizeClasses[size] || sizeClasses.medium;

  return (
    <span
      className={`inline-flex items-center rounded-[2px] ${config.bgColor} ${config.textColor} font-medium ${sizeClass.container} ring-[0.5px] ring-inset ${config.ringColor}`}
    >
      <Icon
        className={`${config.iconColor}`}
        size={sizeClass.icon}
        strokeWidth={1.5}
      />
      HTTP Status: {status}
    </span>
  );
};
