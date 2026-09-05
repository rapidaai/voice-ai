import { Metric } from '@rapidaai/react';
import { getTimeTakenMetric } from '@/utils/metadata';

type RoleVisual = {
  label: string;
  shortLabel: string;
  toneClassName: string;
};

export const getRoleVisual = (role: string): RoleVisual => {
  const normalized = role.toLowerCase();
  if (normalized === 'rapida') {
    return {
      label: 'Assistant',
      shortLabel: 'A',
      toneClassName:
        'bg-blue-100 dark:bg-blue-900/60 text-blue-700 dark:text-blue-300 border-blue-200 dark:border-blue-800',
    };
  }

  if (normalized === 'assistant') {
    return {
      label: 'Assistant',
      shortLabel: 'A',
      toneClassName:
        'bg-teal-100 dark:bg-teal-900/60 text-teal-700 dark:text-teal-300 border-teal-200 dark:border-teal-800',
    };
  }

  if (normalized === 'user') {
    return {
      label: 'User',
      shortLabel: 'U',
      toneClassName:
        'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 border-gray-200 dark:border-gray-700',
    };
  }

  return {
    label: role || 'Unknown',
    shortLabel: (role || '?').slice(0, 1).toUpperCase(),
    toneClassName:
      'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 border-gray-200 dark:border-gray-700',
  };
};

export const formatLatency = (metrics: Metric[]): string => {
  const ns = getTimeTakenMetric(metrics);
  if (ns <= 0) {
    return '0 ms';
  }

  const ms = ns / 1_000_000;
  if (ms < 1000) {
    if (ms < 10) {
      return `${ms.toFixed(2)} ms`;
    }
    return `${ms.toFixed(1)} ms`;
  }

  return `${(ms / 1000).toFixed(2)} s`;
};
