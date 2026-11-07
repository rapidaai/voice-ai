import { memo } from 'react';

export const TextImage = memo(
  (props: { size?: 4 | 5 | 6 | 7 | 8 | 9 | 10; name: string }) => {
    const dimensionClz = (size?: number) => {
      if (size === 10) {
        return 'w-10 h-10';
      }
      if (size === 7) {
        return 'w-7 h-7 text-sm!';
      }

      if (size === 8) {
        return 'w-8 h-8 text-sm!';
      }

      if (size === 6) {
        return 'w-6 h-6 text-sm!';
      }
      if (size === 5) {
        return 'w-5 h-5 text-sm!';
      }
      if (size === 4) {
        return 'w-4 h-4 text-xs!';
      }

      return 'h-9 w-9';
    };

    const randomBgColor = () => {
      return [
        'bg-support-error text-support-error ',
        'bg-support-error text-support-error ',
        'bg-support-success text-support-success ',
        'bg-support-info text-support-info ',
        'bg-data-1 text-data-1 ',
        'bg-support-warning text-support-warning ',
        'bg-support-warning text-support-warning ',
        'bg-support-warning text-support-warning ',
        'bg-support-success text-support-success ',
        'bg-support-info text-support-info ',
        'bg-support-success text-support-success ',
        'bg-support-info text-support-info ',
        'bg-data-1 text-data-1 ',
        'bg-data-1 text-data-1 ',
        'bg-data-5 text-data-5 ',
        'bg-data-5 text-data-5 ',
      ].find((_, i, ar) => Math.random() < 1 / (ar.length - i));
    };

    return (
      <div
        className={`${dimensionClz(
          props.size,
        )} ${randomBgColor()} rounded-[2px] uppercase flex justify-center items-center text-center text-lg shrink-0`}
      >
        <span>{props.name.charAt(0)}</span>
      </div>
    );
  },
);
