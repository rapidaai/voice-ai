import { cn } from '@/styles/media';

/**
 *  */
interface InputHelperProp extends React.LabelHTMLAttributes<HTMLLabelElement> {}

/**
 *
 * @param props
 * @returns
 */
export function InputHelper(props: InputHelperProp) {
  return (
    <div
      className={cn(
        '-mt-1.5 mb-3 text-sm dark:text-gray-500 text-gray-500',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}
