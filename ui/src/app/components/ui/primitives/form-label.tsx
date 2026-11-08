import { cn } from '@/utils';

interface FormLabelProp extends React.LabelHTMLAttributes<HTMLLabelElement> {}

/**
 * Carbon label-01: 12px / 0.75rem, Regular 400, line-height 16px, letter-spacing 0.32px.
 * Used for all field labels in form components.
 */
export function FormLabel(props: FormLabelProp) {
  return (
    <label
      htmlFor={props.htmlFor}
      onClick={props.onClick}
      className={cn(
        // Carbon label-01
        'text-xs font-medium leading-4 tracking-[0.02em]',
        'text-muted ',
        'cursor-pointer',
        props.className,
      )}
    >
      {props.children}
    </label>
  );
}
