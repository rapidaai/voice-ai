import { cn } from '@/styles/media';

export function TableCell(props: React.TdHTMLAttributes<HTMLTableCellElement>) {
  return (
    <td
      {...props}
      className={cn(
        'whitespace-no-wrap p-0 m-0 px-2 md:px-5 py-2',
        props.className,
      )}
    >
      {props.children}
    </td>
  );
}
