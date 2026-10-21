import type {
  ReactNode,
  ChangeEvent,
  FormEvent,
  ForwardRefExoticComponent,
  KeyboardEventHandler,
  MouseEvent,
  RefAttributes,
} from 'react';
import { forwardRef } from 'react';
import {
  Form as CarbonForm,
  Stack as CarbonStack,
  Checkbox as CarbonCheckbox,
  CheckboxSkeleton as CarbonCheckboxSkeleton,
  FormGroup as CarbonFormGroup,
  FluidTextInput as CarbonFluidTextInput,
  TextInput as CarbonTextInput,
  TextInputSkeleton as CarbonTextInputSkeleton,
  TextArea as CarbonTextArea,
  TextAreaSkeleton as CarbonTextAreaSkeleton,
} from '@carbon/react';
import { cn } from '@/utils';

type InputSize = 'sm' | 'md' | 'lg' | 'xl';
type StackOrientation = 'horizontal' | 'vertical';

export interface CarbonFormProps {
  children: ReactNode;
  className?: string;
  onSubmit?: (e: FormEvent<HTMLFormElement>) => void;
}

/** Carbon Form: standard form wrapper with Carbon spacing. */
export function Form({ children, className, onSubmit }: CarbonFormProps) {
  return (
    <CarbonForm className={cn(className)} onSubmit={onSubmit}>
      {children}
    </CarbonForm>
  );
}

export interface CarbonStackProps {
  children: ReactNode;
  className?: string;
  gap: number;
  orientation?: StackOrientation;
}

/** Carbon Stack: layout utility for consistent spacing between elements. */
export function Stack({
  children,
  className,
  gap,
  orientation = 'vertical',
}: CarbonStackProps) {
  return (
    <CarbonStack className={cn(className)} gap={gap} orientation={orientation}>
      {children}
    </CarbonStack>
  );
}

export interface CarbonFormGroupProps {
  children: ReactNode;
  legendText: ReactNode;
  className?: string;
  disabled?: boolean;
  invalid?: boolean;
  message?: boolean;
  messageText?: string;
}

/** Carbon FormGroup: fieldset wrapper with a legend label. */
export function FormGroup({
  children,
  legendText,
  className,
  disabled = false,
  invalid = false,
  message = false,
  messageText,
}: CarbonFormGroupProps) {
  return (
    <CarbonFormGroup
      className={cn(className)}
      legendText={legendText}
      disabled={disabled}
      invalid={invalid}
      message={message}
      messageText={messageText}
    >
      {children}
    </CarbonFormGroup>
  );
}

export interface CarbonTextInputProps {
  id: string;
  labelText: ReactNode;
  className?: string;
  defaultValue?: string | number;
  disabled?: boolean;
  helperText?: ReactNode;
  hideLabel?: boolean;
  inline?: boolean;
  invalid?: boolean;
  invalidText?: ReactNode;
  name?: string;
  onChange?: (e: ChangeEvent<HTMLInputElement>) => void;
  onClick?: (e: MouseEvent<HTMLElement>) => void;
  onKeyDown?: KeyboardEventHandler<HTMLInputElement>;
  placeholder?: string;
  readOnly?: boolean;
  required?: boolean;
  size?: InputSize;
  type?: string;
  value?: string | number;
  warn?: boolean;
  warnText?: ReactNode;
  enableCounter?: boolean;
  maxCount?: number;
  autoComplete?: string;
  'data-testid'?: string;
}

/** Carbon TextInput: single-line text field with label, helper, and validation. */
export const TextInput = forwardRef<HTMLInputElement, CarbonTextInputProps>(
  ({ id, labelText, className, size = 'md', ...rest }, ref) => {
    return (
      <CarbonTextInput
        ref={ref}
        id={id}
        labelText={labelText}
        className={cn(className)}
        size={size}
        {...rest}
      />
    );
  },
);

export type CarbonFluidTextInputProps = Omit<
  CarbonTextInputProps,
  'helperText' | 'hideLabel' | 'inline' | 'size'
>;

type FluidTextInputComponent = ForwardRefExoticComponent<
  CarbonFluidTextInputProps & {
    isPassword?: boolean;
  } & RefAttributes<HTMLInputElement>
>;

const TypedCarbonFluidTextInput =
  CarbonFluidTextInput as FluidTextInputComponent;

/** Carbon FluidTextInput: full-width field row for fluid forms and panels. */
export const FluidTextInput = forwardRef<
  HTMLInputElement,
  CarbonFluidTextInputProps
>(({ className, ...rest }, ref) => (
  <TypedCarbonFluidTextInput ref={ref} className={cn(className)} {...rest} />
));

/** Carbon FluidPasswordInput: fluid field row with password visibility control. */
export const FluidPasswordInput = forwardRef<
  HTMLInputElement,
  CarbonFluidTextInputProps
>(({ className, ...rest }, ref) => (
  <TypedCarbonFluidTextInput
    ref={ref}
    className={cn(className)}
    isPassword
    {...rest}
  />
));

/** Carbon TextInputSkeleton: loading placeholder for TextInput. */
export function TextInputSkeleton({
  className,
  hideLabel = false,
}: {
  className?: string;
  hideLabel?: boolean;
}) {
  return (
    <CarbonTextInputSkeleton className={cn(className)} hideLabel={hideLabel} />
  );
}

export interface CarbonTextAreaProps {
  labelText: ReactNode;
  className?: string;
  cols?: number;
  defaultValue?: string | number;
  disabled?: boolean;
  helperText?: ReactNode;
  hideLabel?: boolean;
  id?: string;
  invalid?: boolean;
  invalidText?: ReactNode;
  name?: string;
  onChange?: (e: ChangeEvent<HTMLTextAreaElement>) => void;
  placeholder?: string;
  readOnly?: boolean;
  required?: boolean;
  rows?: number;
  value?: string | number;
  warn?: boolean;
  warnText?: ReactNode;
  enableCounter?: boolean;
  maxCount?: number;
}

/** Carbon TextArea: multi-line text field with label, helper, and validation. */
export function TextArea({
  labelText,
  className,
  rows = 4,
  ...rest
}: CarbonTextAreaProps) {
  return (
    <CarbonTextArea
      labelText={labelText}
      className={cn(className)}
      rows={rows}
      {...rest}
    />
  );
}

/** Carbon TextAreaSkeleton: loading placeholder for TextArea. */
export function TextAreaSkeleton({
  className,
  hideLabel = false,
}: {
  className?: string;
  hideLabel?: boolean;
}) {
  return (
    <CarbonTextAreaSkeleton className={cn(className)} hideLabel={hideLabel} />
  );
}

export interface CarbonCheckboxProps {
  id: string;
  labelText: string;
  className?: string;
  checked?: boolean;
  defaultChecked?: boolean;
  disabled?: boolean;
  helperText?: ReactNode;
  hideLabel?: boolean;
  indeterminate?: boolean;
  invalid?: boolean;
  invalidText?: ReactNode;
  name?: string;
  onChange?: (
    e: ChangeEvent<HTMLInputElement>,
    data: { checked: boolean; id: string },
  ) => void;
  onClick?: (e: MouseEvent<HTMLInputElement>) => void;
  warn?: boolean;
  warnText?: ReactNode;
}

/** Carbon Checkbox: single checkbox with label and validation support. */
export function Checkbox({
  id,
  labelText,
  className,
  ...rest
}: CarbonCheckboxProps) {
  return (
    <CarbonCheckbox
      id={id}
      labelText={labelText}
      className={cn(className)}
      {...rest}
    />
  );
}

/** Carbon CheckboxSkeleton: loading placeholder for Checkbox. */
export function CheckboxSkeleton({ className }: { className?: string }) {
  return <CarbonCheckboxSkeleton className={cn(className)} />;
}
