import { cn } from '@/styles/media';

export function TableRow(props: React.HTMLAttributes<HTMLTableRowElement>) {
  return (
    <tr
      {...props}
      className={cn(
        'dark:border-gray-800 border-gray-300 border-b-[0.5px] hover:bg-gray-50 dark:hover:bg-gray-950/20 text-sm',
        props.className,
      )}
    >
      {props.children}
    </tr>
  );
}
