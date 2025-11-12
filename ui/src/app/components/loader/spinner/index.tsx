import { cn } from '@/utils';

interface SpinnerProps {
  size?: 'xs' | 'sm' | 'md' | 'lg';
  className?: string;
}

export function Spinner({ size = 'sm', className }: SpinnerProps) {
  return (
    // <div role="status" className={className}>
    <div
      className={cn(
        size === 'md' && 'h-9 w-9 border-[2.5px]',
        size === 'lg' && 'h-14 w-14 border-[2.5px]',
        size === 'sm' && 'h-5 w-5 border-2',
        size === 'xs' && 'h-[13px] w-[13px] border-2',
        'rounded-full border-blue-600 border-r-transparent! border-b-blue-900 animate-spin3s',
        className,
      )}
      style={{
        borderTopStyle: 'solid',
        borderLeftStyle: 'solid',
        borderBottomStyle: 'dotted',
      }}
    ></div>
  );
}
