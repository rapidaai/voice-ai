import { Metric } from '@rapidaai/react';
import { BlueNoticeBlock } from '@/app/components/container/message/notice-block';
import { Tooltip } from '@/app/components/Tooltip';
import { cn } from '@/styles/media';
import { InfoIcon } from 'lucide-react';
import { FC } from 'react';

export const MessageMetrics: FC<{ metrics: Array<Metric> }> = ({ metrics }) => {
  if (metrics.length <= 0)
    return (
      <BlueNoticeBlock>
        There are no metrics recorded for given message.
      </BlueNoticeBlock>
    );
  return (
    <div className="grid grid-cols-4 gap-2 m-4">
      {metrics.map((x, idx) => {
        return (
          <div
            className="flex justify-between items-center border-[0.5px] rounded-[2px]"
            key={`metrics-idx-${idx}`}
          >
            <div className="py-3 px-4 flex items-center gap-2 font-mono lowercase truncate">
              <span className="truncate">{x.getName()}</span>
              <Tooltip
                icon={
                  <InfoIcon className="w-4 h-4 mt-[2px] ml-0.5 opacity-50dark:text-gray-400 shrink-0" />
                }
              >
                <p className={cn('font-normal text-sm p-1 w-fit px-2')}>
                  {x.getDescription()}
                </p>
              </Tooltip>
            </div>
            <div className="py-3 px-4 ">
              <div className="flex items-center">
                <MetricValue value={x.getValue()} />
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
};
const MetricValue = ({ value }) => {
  if (
    typeof value === 'string' &&
    !isNaN(parseFloat(value)) &&
    parseFloat(value) < 1
  ) {
    // If value is a string but a valid number (quantifiable), parse it to percentage
    const progress = Math.min(Math.max(parseFloat(value), 0), 100); // Clamp value between 0 and 100
    return (
      <div className="w-full bg-gray-200 rounded-[2px] h-2.5 dark:bg-gray-700">
        <div
          className="bg-blue-600 h-2.5 rounded-[2px]"
          style={{
            width: `${progress}%`,
          }}
        ></div>
      </div>
    );
  } else if (value == 'false') {
    // If value is a boolean, show a checkmark for true
    return (
      <svg
        viewBox="0 0 24 24"
        fill="currentColor"
        stroke="none"
        className="w-7 h-7 text-green-500"
      >
        <path d="M17 3.34a10 10 0 1 1 -14.995 8.984l-.005 -.324l.005 -.324a10 10 0 0 1 14.995 -8.336zm-1.293 5.953a1 1 0 0 0 -1.32 -.083l-.094 .083l-3.293 3.292l-1.293 -1.292l-.094 -.083a1 1 0 0 0 -1.403 1.403l.083 .094l2 2l.094 .083a1 1 0 0 0 1.226 0l.094 -.083l4 -4l.083 -.094a1 1 0 0 0 -.083 -1.32z"></path>
      </svg>
    );
  } else if (value == 'true') {
    return (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="currentColor"
        stroke="none"
        className="w-7 h-7 text-red-500"
      >
        <path d="M17 3.34a10 10 0 1 1 -14.995 8.984l-.005 -.324l.005 -.324a10 10 0 0 1 14.995 -8.336zm-6.489 5.8a1 1 0 0 0 -1.218 1.567l1.292 1.293l-1.292 1.293l-.083 .094a1 1 0 0 0 1.497 1.32l1.293 -1.292l1.293 1.292l.094 .083a1 1 0 0 0 1.32 -1.497l-1.292 -1.293l1.292 -1.293l.083 -.094a1 1 0 0 0 -1.497 -1.32l-1.293 1.292l-1.293 -1.292l-.094 -.083z"></path>
      </svg>
    );
  } else {
    // If value is a non-quantifiable string, just display it as text
    return <span>{value}</span>;
  }
};
