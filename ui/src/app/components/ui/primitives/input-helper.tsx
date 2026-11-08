import { cn } from '@/utils';

interface InputHelperProp extends React.HTMLAttributes<HTMLParagraphElement> {}

/**
 * Carbon helper-text-01: 12px / 0.75rem, Regular 400, line-height 16px, letter-spacing 0.32px.
 * Always placed directly below the field it describes.
 */
export function InputHelper(props: InputHelperProp) {
  return (
    <p
      className={cn(
        // Carbon helper-text-01
        'text-xs leading-4 tracking-[0.02em]',
        'text-muted',
        props.className,
      )}
    >
      {props.children}
    </p>
  );
}
