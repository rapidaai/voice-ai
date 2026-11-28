import type { ReactNode } from 'react';
import {
  ComboBox as CarbonComboBox,
  Dropdown as CarbonDropdown,
  DropdownSkeleton as CarbonDropdownSkeleton,
  MultiSelect as CarbonMultiSelect,
} from '@carbon/react';
import { cn } from '@/utils';

// Types

type DropdownSize = 'sm' | 'md' | 'lg';
type DropdownDirection = 'top' | 'bottom';

// ComboBox (searchable single select)

export interface CarbonComboBoxProps<T> {
  id: string;
  'aria-label'?: string;
  titleText: ReactNode;
  placeholder: string;
  items: T[];
  itemToString?: (item: T | null) => string;
  selectedItem?: T | null;
  initialSelectedItem?: T | null;
  onChange?: (data: { selectedItem: T | null }) => void;
  className?: string;
  size?: DropdownSize;
  direction?: DropdownDirection;
  disabled?: boolean;
  readOnly?: boolean;
  invalid?: boolean;
  invalidText?: ReactNode;
  warn?: boolean;
  warnText?: ReactNode;
  helperText?: ReactNode;
}

/** Carbon ComboBox: searchable single-select dropdown list. */
export function ComboBox<T>({
  id,
  'aria-label': ariaLabel,
  titleText,
  placeholder,
  items,
  itemToString,
  selectedItem,
  initialSelectedItem,
  onChange,
  className,
  size = 'md',
  direction = 'bottom',
  disabled = false,
  readOnly = false,
  invalid = false,
  invalidText,
  warn = false,
  warnText,
  helperText,
}: CarbonComboBoxProps<T>) {
  return (
    <CarbonComboBox
      id={id}
      aria-label={ariaLabel}
      titleText={titleText}
      placeholder={placeholder}
      items={items}
      itemToString={
        itemToString || ((item: T | null) => (item ? String(item) : ''))
      }
      selectedItem={selectedItem}
      initialSelectedItem={initialSelectedItem ?? undefined}
      onChange={data => onChange?.({ selectedItem: data.selectedItem ?? null })}
      className={cn(className)}
      size={size}
      direction={direction}
      disabled={disabled}
      readOnly={readOnly}
      invalid={invalid}
      invalidText={invalidText}
      warn={warn}
      warnText={warnText}
      helperText={helperText}
    />
  );
}

// Dropdown (single select)

export interface CarbonDropdownProps<T> {
  id: string;
  autoAlign?: boolean;
  'aria-label'?: string;
  titleText: ReactNode;
  label: string;
  items: T[];
  itemToString?: (item: T | null) => string;
  selectedItem?: T | null;
  initialSelectedItem?: T | null;
  onChange?: (data: { selectedItem: T | null }) => void;
  className?: string;
  size?: DropdownSize;
  direction?: DropdownDirection;
  disabled?: boolean;
  readOnly?: boolean;
  invalid?: boolean;
  invalidText?: ReactNode;
  warn?: boolean;
  warnText?: ReactNode;
  helperText?: ReactNode;
  hideLabel?: boolean;
  type?: 'default' | 'inline';
  isLoading?: boolean;
}

/** Carbon Dropdown: single-select dropdown list. */
export function Dropdown<T>({
  id,
  autoAlign,
  'aria-label': ariaLabel,
  titleText,
  label,
  items,
  itemToString,
  selectedItem,
  initialSelectedItem,
  onChange,
  className,
  size = 'md',
  direction = 'bottom',
  disabled = false,
  readOnly = false,
  invalid = false,
  invalidText,
  warn = false,
  warnText,
  helperText,
  hideLabel = false,
  type = 'default',
  isLoading = false,
}: CarbonDropdownProps<T>) {
  if (isLoading) {
    return (
      <div className={cn(className)}>
        <CarbonDropdownSkeleton size={size} hideLabel={hideLabel} />
      </div>
    );
  }

  return (
    <CarbonDropdown
      id={id}
      autoAlign={autoAlign}
      aria-label={ariaLabel}
      titleText={titleText}
      label={label}
      items={items}
      itemToString={
        itemToString || ((item: T | null) => (item ? String(item) : ''))
      }
      selectedItem={selectedItem}
      initialSelectedItem={initialSelectedItem}
      onChange={onChange}
      className={cn(className)}
      size={size}
      direction={direction}
      disabled={disabled}
      readOnly={readOnly}
      invalid={invalid}
      invalidText={invalidText}
      warn={warn}
      warnText={warnText}
      helperText={helperText}
      hideLabel={hideLabel}
      type={type}
    />
  );
}

// MultiSelect

export interface CarbonMultiSelectProps<T> {
  id: string;
  autoAlign?: boolean;
  titleText: ReactNode;
  label: string;
  items: T[];
  itemToString?: (item: T) => string;
  selectedItems?: T[];
  initialSelectedItems?: T[];
  onChange?: (data: { selectedItems: T[] | null }) => void;
  className?: string;
  size?: DropdownSize;
  direction?: DropdownDirection;
  disabled?: boolean;
  readOnly?: boolean;
  invalid?: boolean;
  invalidText?: ReactNode;
  warn?: boolean;
  warnText?: ReactNode;
  helperText?: ReactNode;
  hideLabel?: boolean;
  type?: 'default' | 'inline';
  selectionFeedback?: 'fixed' | 'top' | 'top-after-reopen';
  clearSelectionText?: string;
  clearSelectionDescription?: string;
}

/** Carbon MultiSelect: multi-select dropdown with checkboxes. */
export function MultiSelect<T>({
  id,
  autoAlign,
  titleText,
  label,
  items,
  itemToString,
  selectedItems,
  initialSelectedItems,
  onChange,
  className,
  size = 'md',
  direction = 'bottom',
  disabled = false,
  readOnly = false,
  invalid = false,
  invalidText,
  warn = false,
  warnText,
  helperText,
  hideLabel = false,
  type = 'default',
  selectionFeedback = 'top-after-reopen',
  clearSelectionText = 'Clear all',
  clearSelectionDescription = 'Total items selected:',
}: CarbonMultiSelectProps<T>) {
  return (
    <CarbonMultiSelect
      id={id}
      autoAlign={autoAlign}
      titleText={titleText}
      label={label}
      items={items}
      itemToString={itemToString || ((item: T) => (item ? String(item) : ''))}
      selectedItems={selectedItems}
      initialSelectedItems={initialSelectedItems}
      onChange={onChange}
      className={cn(className)}
      size={size}
      direction={direction}
      disabled={disabled}
      readOnly={readOnly}
      invalid={invalid}
      invalidText={invalidText}
      warn={warn}
      warnText={warnText}
      helperText={helperText}
      hideLabel={hideLabel}
      type={type}
      selectionFeedback={selectionFeedback}
      clearSelectionText={clearSelectionText}
      clearSelectionDescription={clearSelectionDescription}
    />
  );
}
