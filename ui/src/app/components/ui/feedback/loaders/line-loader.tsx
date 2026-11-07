import { useRapidaStore } from '@/stores/app';
import type { HTMLAttributes } from 'react';

import { cn } from '@/utils';

export function LineLoader(props: HTMLAttributes<HTMLDivElement>) {
  const { loading } = useRapidaStore();
  return <AnimatedLine animate={loading ? 'infinite' : ''} {...props} />;
}

export function AnimatedLine({
  animate,
  className,
  ...props
}: HTMLAttributes<HTMLDivElement> & { animate: string }) {
  return (
    <div {...props} className={cn('w-full overflow-hidden', className)}>
      <div
        className={cn(
          'bg-linear-to-r h-0.5 from-data-1 via-data-1 to-data-5 w-full transition-all duration-700 translate-x-full',
        )}
        style={{
          animation: `fill 2s linear ${animate}`,
        }}
      ></div>
    </div>
  );
}
