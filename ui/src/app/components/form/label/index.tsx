import { cn } from '@/utils';

/**
 *  */
interface LabelProp extends React.LabelHTMLAttributes<HTMLLabelElement> {
  for?: string;
  text?: any;
}

/**
 *
 * @param props
 * @returns
 */
export function Label(props: LabelProp) {
  return (
    <label
      htmlFor={props.for}
      className={cn(
        'leading-6 cursor-pointer inline-flex items-center capitalize font-medium text-[0.95rem] dark:text-base',
        props.className,
      )}
      onClick={props.onClick}
    >
      {props.text || props.children}
    </label>
  );
}
