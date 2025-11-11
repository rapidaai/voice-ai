import { CodeEditor } from '@/app/components/Form/editor/code-editor';
import { cn } from '@/styles/media';
import React, { HTMLAttributes, useState } from 'react';

interface TextAreaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  row?: number;
  wrapperClassName?: string;
}

/**
 *
 */
export const Textarea = React.forwardRef<HTMLTextAreaElement, TextAreaProps>(
  (props: TextAreaProps, ref) => {
    /**
     * when any request is going disable all the input boxes
     */
    return (
      <textarea
        {...props}
        id={props.name}
        ref={ref}
        required={props.required}
        name={props.name}
        rows={props.row}
        className={cn(
          'block p-2.5 resize-none w-full',
          'dark:placeholder-gray-600 placeholder-gray-400',
          'outline-solid outline-[1.5px] outline-transparent',
          'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
          'border-b border-gray-400 dark:border-gray-600',
          'dark:focus:border-blue-600 focus:border-blue-600',
          'transition-all duration-200 ease-in-out',
          'dark:text-gray-300 text-gray-600',
          'bg-white dark:bg-gray-950',
          props.className,
        )}
        placeholder={props.placeholder}
      ></textarea>
    );
  },
);

interface TextAreaWithActionProps extends TextAreaProps {
  actions?: React.ReactElement;
}
export const ScalableTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaWithActionProps
>((props: TextAreaWithActionProps, ref) => {
  /**
   * when any request is going disable all the input boxes
   */
  const [textareaHeight, setTextareaHeight] = useState('auto');
  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    // Reset height to auto to allow shrinking
    e.target.style.height = 'auto';
    // Set height to scroll height, with a minimum of 32px (adjust as needed)
    e.target.style.height = `${Math.max(e.target.scrollHeight, 32)}px`;

    // Update the state if needed
    if (textareaHeight !== e.target.style.height) {
      setTextareaHeight(e.target.style.height);
    }

    // Propagate onChange event to parent component
    if (props.onChange) {
      props.onChange(e);
    }
  };
  const { wrapperClassName, ...attr } = props;
  return (
    <div
      className={cn(
        'block p-2.5 resize-none w-full',
        'dark:placeholder-gray-600 placeholder-gray-400',
        'outline-solid outline-[1.5px] outline-transparent',
        'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
        'border-b border-gray-400 dark:border-gray-600',
        'dark:focus:border-blue-600 focus:border-blue-600',
        'transition-all duration-200 ease-in-out',
        'dark:text-gray-300 text-gray-600',
        'bg-white dark:bg-gray-950',
        wrapperClassName,
      )}
    >
      <textarea
        {...attr}
        id={props.name}
        ref={ref}
        required
        name={props.name}
        onChange={handleChange}
        style={{ height: textareaHeight }} // Dynamically set height
        className={cn(
          'p-2',
          'block resize-none w-full min-h-12 max-h-80',
          'dark:placeholder-gray-600 placeholder-gray-400',
          'focus:ring-0 focus:outline-hidden',
          'bg-white dark:bg-gray-950',
          'focus:bg-white',
          props.className,
        )}
        rows={props.row}
        placeholder={props.placeholder}
      ></textarea>
      {props.actions && props.actions}
    </div>
  );
});

/**
 * for paragraph you can do what magic you would want to do in future
 */
export const ParagraphTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaProps
>((attr, ref) => {
  return (
    <ScalableTextarea
      ref={ref}
      placeholder="Enter variable value..."
      spellCheck="false"
      className="form-input px-2"
      wrapperClassName={`
        border-transparent!
        outline-hidden!
        shadow-none!
        bg-transparent
      `}
      {...attr}
      required
    />
  );
});

export const NumberTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaProps
>((attr, ref) => {
  return (
    <ScalableTextarea
      ref={ref}
      placeholder="Enter variable value..."
      spellCheck="false"
      className="form-input px-2"
      wrapperClassName={`
        border-transparent!
        outline-hidden!
        shadow-none!
        bg-transparent
      `}
      {...attr}
      required
    />
  );
});

export const UrlTextarea = React.forwardRef<HTMLTextAreaElement, TextAreaProps>(
  (attr, ref) => {
    return (
      <ScalableTextarea
        ref={ref}
        placeholder="Enter variable value..."
        spellCheck="false"
        className="form-input px-2"
        wrapperClassName={`
        border-transparent!
        outline-hidden!
        shadow-none!
        bg-transparent
      `}
        {...attr}
        required
      />
    );
  },
);

export const TextTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaProps
>((attr, ref) => {
  return (
    <ScalableTextarea
      ref={ref}
      placeholder="Enter variable value..."
      spellCheck="false"
      className="form-input"
      wrapperClassName={`
        p-0 
        border-transparent!
        outline-hidden!
        shadow-none!
        bg-transparent
      `}
      {...attr}
      required
    />
  );
});
export const JsonTextarea = React.forwardRef<
  HTMLTextAreaElement,
  TextAreaProps
>((props, ref) => {
  const { children, onChange, className } = props;

  return (
    <CodeEditor
      placeholder="Provide a tool parameters that will be passed to llm"
      value={children as string}
      onChange={value =>
        onChange &&
        onChange({
          target: { value },
        } as React.ChangeEvent<HTMLTextAreaElement>)
      }
      className={cn(
        'min-h-40 max-h-dvh bg-light-background dark:bg-gray-950',
        className,
      )}
    />
  );
});
