import React from 'react';
import { UserCog, Shield, Edit3, BookOpen, User } from 'lucide-react';

export const RoleIndicator = ({ role, size = 'medium' }) => {
  const roleConfig = {
    SUPER_ADMIN: {
      bgColor: 'bg-purple-100/50 dark:bg-purple-900/30',
      textColor: 'text-purple-800/80 dark:text-purple-200/60',
      Icon: UserCog,
      display: 'Super Admin',
    },
    ADMIN: {
      bgColor: 'bg-blue-100/50 dark:bg-blue-900/30',
      textColor: 'text-blue-800/80 dark:text-blue-200/60',
      Icon: Shield,
      display: 'Admin',
    },
    WRITER: {
      bgColor: 'bg-green-100/50 dark:bg-green-900/30',
      textColor: 'text-green-800/80 dark:text-green-200/60',
      Icon: Edit3,
      display: 'Writer',
    },
    READER: {
      bgColor: 'bg-yellow-100/50 dark:bg-yellow-900/30',
      textColor: 'text-yellow-800/80 dark:text-yellow-200/60',
      Icon: BookOpen,
      display: 'Reader',
    },
    DEFAULT: {
      bgColor: 'bg-gray-100/50 dark:bg-gray-800/30',
      textColor: 'text-gray-800/80 dark:text-gray-200/60',
      Icon: User,
      display: 'User',
    },
  };
  const config = roleConfig[role.toUpperCase()] || roleConfig['DEFAULT'];
  const { Icon } = config;

  const sizeClasses = {
    small: {
      container: 'text-xs px-2 py-0.5',
      icon: 12,
    },
    medium: {
      container: 'text-sm px-2.5 py-1',
      icon: 16,
    },
    large: {
      container: 'text-base px-3 py-1.5',
      icon: 18,
    },
  };

  const sizeClass = sizeClasses[size] || sizeClasses.medium;

  return (
    <span
      className={`inline-flex items-center rounded-[2px] ${config.bgColor} ${config.textColor} ${sizeClass.container} gap-1.5`}
    >
      <Icon size={sizeClass.icon} />
      {config.display}
    </span>
  );
};
