import type { Metric } from '@rapidaai/react';
import { Tooltip } from '@/app/components/ui/primitives';
import { ProgressBar } from '@carbon/react';
import { EmptyState } from '@/app/components/ui/feedback';
import {
  ChartLine,
  CheckmarkFilled,
  ErrorFilled,
  Information,
} from '@carbon/icons-react';

export function EndpointMetrics({ metrics }: { metrics: Array<Metric> }) {
  if (metrics.length <= 0)
    return (
      <EmptyState
        className="h-full min-h-[420px]"
        icon={ChartLine}
        title="No metrics found"
        subtitle="No metrics were recorded for this trace."
      />
    );
  return (
    <div className="divide-y divide-border-subtle w-full">
      {metrics.map((metric, index) => (
        <div
          key={`metrics-idx-${index}`}
          className="flex items-center justify-between h-12 px-4 gap-4"
        >
          <div className="flex items-center gap-1.5">
            <span className="text-xs font-medium uppercase tracking-[0.08em] text-muted shrink-0">
              {metric.getName()}
            </span>
            <Tooltip
              icon={<Information size={14} className="text-muted shrink-0" />}
            >
              <p className="font-normal text-sm p-1 px-2">
                {metric.getDescription()}
              </p>
            </Tooltip>
          </div>
          <div className="flex items-center shrink-0 ml-4">
            <MetricValue value={metric.getValue()} />
          </div>
        </div>
      ))}
    </div>
  );
}

interface MetricValueProps {
  value: string;
}

const MetricValue = ({ value }: MetricValueProps) => {
  const numericValue = Number.parseFloat(value);
  if (Number.isFinite(numericValue) && numericValue >= 0 && numericValue <= 1) {
    const progress = Math.round(numericValue * 100);
    return (
      <ProgressBar
        className="w-24"
        hideLabel
        label="Metric progress"
        max={100}
        size="small"
        value={progress}
      />
    );
  }

  if (value === 'false') {
    return <CheckmarkFilled size={20} className="text-support-success" />;
  }

  if (value === 'true') {
    return <ErrorFilled size={20} className="text-support-error" />;
  }

  return <span className="tabular-nums">{value}</span>;
};
