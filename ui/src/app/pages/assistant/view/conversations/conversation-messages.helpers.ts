import { Metric } from '@rapidaai/react';
import { getTimeTakenMetric } from '@/utils/metadata';

type RoleVisual = {
  label: string;
  shortLabel: string;
  toneClassName: string;
};

export const getRoleVisual = (role: string): RoleVisual => {
  const roleKey = role.toLowerCase();
  if (roleKey === 'rapida') {
    return {
      label: 'Assistant',
      shortLabel: 'A',
      toneClassName:
        'bg-support-info/10 text-support-info border-support-info ',
    };
  }

  if (roleKey === 'assistant') {
    return {
      label: 'Assistant',
      shortLabel: 'A',
      toneClassName:
        'bg-support-info/10 text-support-info border-support-info ',
    };
  }

  if (roleKey === 'user') {
    return {
      label: 'User',
      shortLabel: 'U',
      toneClassName: 'bg-layer text-muted border-border-subtle ',
    };
  }

  return {
    label: role || 'Unknown',
    shortLabel: (role || '?').slice(0, 1).toUpperCase(),
    toneClassName: 'bg-layer text-muted border-border-subtle ',
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
