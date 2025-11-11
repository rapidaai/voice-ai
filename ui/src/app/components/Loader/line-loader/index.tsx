import { useRapidaStore } from '@/hooks';

import { cn } from '@/styles/media';

export function LineLoader(props: React.HTMLAttributes<HTMLDivElement>) {
  const { loading } = useRapidaStore();
  return <AnimatedLine animate={loading ? 'infinite' : ''} />;
}

export const AnimatedLine: React.FC<
  React.HTMLAttributes<HTMLDivElement> & { animate: string }
> = ({ animate, ...props }) => {
  return (
    <div className={cn('w-full overflow-hidden', props.className)} {...props}>
      <div
        className={cn(
          'bg-linear-to-r h-0.5 from-indigo-500 via-purple-500 to-pink-500 w-full transition-all duration-700 translate-x-full',
          props.className,
        )}
        style={{
          animation: `fill 2s linear ${animate}`,
        }}
      ></div>
    </div>
  );
};
