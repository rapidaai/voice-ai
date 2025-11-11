import { cn } from '@/styles/media';

/**
 *  */
interface FormLabelProp extends React.LabelHTMLAttributes<HTMLLabelElement> {}

/**
 *
 * @param props
 * @returns
 */
export function FormLabel(props: FormLabelProp) {
  return (
    <label
      htmlFor={props.htmlFor}
      className={cn(
        'leading-6 cursor-pointer inline-flex items-center capitalize text-sm font-medium dark:text-gray-500 text-gray-500',
        props.className,
      )}
      onClick={props.onClick}
    >
      {props.children}
    </label>
  );
}
