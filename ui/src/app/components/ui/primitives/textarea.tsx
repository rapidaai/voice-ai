import { cn } from '@/utils';
import {
  TextArea as CarbonTextArea,
  type TextAreaProps as CarbonTextAreaProps,
} from '@carbon/react';
import React, { useState } from 'react';

export interface TextAreaProps
  extends Omit<CarbonTextAreaProps, 'id' | 'labelText' | 'rows'> {
  id?: string;
  labelText?: CarbonTextAreaProps['labelText'];
  row?: number;
  rows?: number;
  wrapperClassName?: string;
}

export const Textarea = React.forwardRef<HTMLTextAreaElement, TextAreaProps>(
  (
    {
      'aria-label': ariaLabel,
      className,
      hideLabel = true,
      id,
      labelText,
      name,
      row,
      rows,
      ...props
    },
    ref,
  ) => {
    const generatedId = React.useId().replace(/:/g, '');
    const textareaId = id ?? name ?? `textarea-${generatedId}`;
    const accessibleLabel =
      labelText ||
      ariaLabel ||
      (typeof name === 'string' ? name : undefined) ||
      props.placeholder ||
      'Textarea';

    return (
      <CarbonTextArea
        {...props}
        ref={ref}
        id={textareaId}
        name={name}
        rows={rows ?? row ?? 3}
        labelText={accessibleLabel}
        hideLabel={hideLabel}
        className={cn('w-full', className)}
      />
    );
  },
);

interface TextAreaWithActionProps extends TextAreaProps {
  actions?: React.ReactElement;
}

/**
 * Auto-expanding textarea. Grows with content, no scroll. Used in prompt editors.
 */
export const ScalableTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaWithActionProps
>((props, ref) => {
  const [textareaHeight, setTextareaHeight] = useState('auto');

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    e.target.style.height = 'auto';
    e.target.style.height = `${Math.max(e.target.scrollHeight, 32)}px`;
    if (textareaHeight !== e.target.style.height) {
      setTextareaHeight(e.target.style.height);
    }
    if (props.onChange) props.onChange(e);
  };

  const { wrapperClassName, actions, ...attr } = props;

  return (
    <div
      className={cn(
        'block w-full px-4 py-2.5',
        'bg-light-background bg-surface',
        'border-0 border-b border-border-subtle ',
        'outline-solid outline-[1.5px] outline-transparent outline-offset-[-1.5px]',
        'focus-within:outline-primary focus-within:border-primary',
        'rounded-none transition-colors duration-100',
        wrapperClassName,
      )}
    >
      <textarea
        {...attr}
        id={props.name}
        ref={ref}
        onChange={handleChange}
        style={{ height: textareaHeight }}
        className={cn(
          'block w-full resize-none min-h-8 max-h-80',
          'text-sm text-foreground ',
          'placeholder-placeholder',
          'bg-transparent',
          'focus:ring-0 focus:outline-hidden',
          props.className,
        )}
        rows={props.row}
      />
      {actions}
    </div>
  );
});

/** Inline paragraph input. Transparent wrapper, no border. Used in variable editors. */
export const ParagraphTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaProps
>((attr, ref) => (
  <ScalableTextarea
    ref={ref}
    placeholder="Enter variable value..."
    spellCheck="false"
    className="form-input p-2"
    wrapperClassName="border-transparent! outline-hidden! bg-transparent p-0"
    {...attr}
    required
  />
));

export const NumberTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaProps
>((attr, ref) => (
  <ScalableTextarea
    ref={ref}
    placeholder="Enter variable value..."
    spellCheck="false"
    className="form-input p-2"
    wrapperClassName="border-transparent! outline-hidden! bg-transparent p-0"
    {...attr}
    required
  />
));

export const UrlTextarea = React.forwardRef<HTMLTextAreaElement, TextAreaProps>(
  (attr, ref) => (
    <ScalableTextarea
      ref={ref}
      placeholder="Enter variable value..."
      spellCheck="false"
      className="form-input p-2"
      wrapperClassName="border-transparent! outline-hidden! bg-transparent p-0"
      {...attr}
      required
    />
  ),
);

export const TextTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaProps
>((attr, ref) => (
  <ScalableTextarea
    ref={ref}
    placeholder="Enter variable value..."
    spellCheck="false"
    className="form-input p-2"
    wrapperClassName="p-0 border-transparent! outline-hidden! bg-transparent"
    {...attr}
    required
  />
));

export const JsonTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaProps
>((attr, ref) => (
  <ScalableTextarea
    ref={ref}
    placeholder="Enter variable value..."
    spellCheck="false"
    className="form-input p-2"
    wrapperClassName="p-0 border-transparent! outline-hidden! bg-transparent"
    {...attr}
    required
  />
));
