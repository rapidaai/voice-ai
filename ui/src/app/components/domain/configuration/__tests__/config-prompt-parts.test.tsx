import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { PromptRole } from '@/models/prompt';
import { InputVarType } from '@/models/common';
import MessageTypeSelector from '../config-prompt/message-type-selector';
import AdvancedPromptInput from '../config-prompt/advanced-prompt-input';
import { TypeOfVariable } from '../config-prompt/type-of-variable';
import ConfigSelect from '../config-var/config-select';

jest.mock('@carbon/icons-react', () => ({
  Add: () => <svg data-testid="add-icon" />,
  Checkmark: () => <svg data-testid="checkmark-icon" />,
  Copy: () => <svg data-testid="copy-icon" />,
  Draggable: () => <svg data-testid="drag-icon" />,
  Maximize: () => <svg data-testid="maximize-icon" />,
  Minimize: () => <svg data-testid="minimize-icon" />,
  TrashCan: () => <svg data-testid="trash-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Button: ({
    children,
    className: _className,
    hasIconOnly: _hasIconOnly,
    iconDescription,
    kind: _kind,
    renderIcon: Icon,
    size: _size,
    ...props
  }: any) => (
    <button aria-label={iconDescription} type="button" {...props}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
  Dropdown: ({
    className,
    id,
    itemToString,
    items = [],
    label,
    onChange,
    selectedItem,
  }: any) => (
    <select
      aria-label={label}
      className={className}
      id={id}
      value={selectedItem ?? ''}
      onChange={event => {
        const selected = items.find(
          (item: string) => item === event.target.value,
        );
        onChange?.({ selectedItem: selected ?? null });
      }}
    >
      <option value="">Select</option>
      {items.map((item: string) => (
        <option key={item} value={item}>
          {itemToString(item)}
        </option>
      ))}
    </select>
  ),
}));

jest.mock('@/app/components/ui/primitives', () => ({
  GhostButton: ({ children, iconDescription, onClick, tabIndex }: any) => (
    <button
      type="button"
      aria-label={iconDescription}
      tabIndex={tabIndex}
      onClick={onClick}
    >
      {children}
    </button>
  ),
  Select: ({
    'aria-label': ariaLabel,
    id,
    onChange,
    options,
    placeholder,
    value,
  }: any) => (
    <select
      id={id}
      aria-label={ariaLabel ?? placeholder}
      value={value}
      onChange={onChange}
    >
      {placeholder ? <option value="">{placeholder}</option> : null}
      {options.map((option: { name: string; value: string }) => (
        <option key={option.value} value={option.value}>
          {option.name}
        </option>
      ))}
    </select>
  ),
  TertiaryButton: ({ children, onClick, renderIcon: Icon }: any) => (
    <button type="button" onClick={onClick}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
  TextInput: ({ id, onChange, placeholder, value }: any) => (
    <input
      aria-label={id}
      id={id}
      placeholder={placeholder}
      value={value}
      onChange={onChange}
    />
  ),
}));

jest.mock('@/app/components/domain/prompt-editor/prompt-editor', () => ({
  __esModule: true,
  default: ({ onBlur, onChange, onFocus, value }: any) => (
    <textarea
      aria-label="Prompt editor"
      value={value}
      onBlur={onBlur}
      onChange={event => onChange(event.target.value)}
      onFocus={onFocus}
    />
  ),
}));

jest.mock('react-sortablejs', () => ({
  ReactSortable: ({ children, list, setList }: any) => (
    <tbody>
      {children}
      <tr>
        <td colSpan={3}>
          <button type="button" onClick={() => setList([...list].reverse())}>
            Reverse options
          </button>
        </td>
      </tr>
    </tbody>
  ),
}));

describe('configuration prompt subcomponents', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    Object.assign(navigator, {
      clipboard: {
        writeText: jest.fn(),
      },
    });
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('selects prompt message roles with Carbon dropdown behavior', () => {
    const onChange = jest.fn();

    render(<MessageTypeSelector value={PromptRole.user} onChange={onChange} />);

    const selector = screen.getByRole('combobox', {
      name: 'Select a role',
    });
    expect(selector).toHaveValue(PromptRole.user);
    expect(selector).toHaveClass('w-full');
    expect(selector).not.toHaveClass('[&_.cds--list-box]:!border-none');

    fireEvent.change(selector, { target: { value: PromptRole.assistant } });
    fireEvent.change(selector, { target: { value: '' } });

    expect(onChange).toHaveBeenCalledWith(PromptRole.assistant);
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it('changes variable type from the configured option list', () => {
    const onChange = jest.fn();

    render(
      <TypeOfVariable
        id="argument-type-topic"
        aria-label="Type for topic"
        hideLabel
        labelText="Type for topic"
        type={InputVarType.textInput}
        allType={[InputVarType.textInput, InputVarType.number]}
        onChange={onChange}
      />,
    );

    const selector = screen.getByLabelText('Type for topic');

    expect(selector).toHaveAttribute('id', 'argument-type-topic');
    fireEvent.change(selector, {
      target: { value: InputVarType.number },
    });

    expect(onChange).toHaveBeenCalledWith(InputVarType.number);
  });

  it('edits, adds, removes, and reorders config select options', () => {
    const onChange = jest.fn();

    render(
      <ConfigSelect
        placeholder="Add option"
        label="Add choice"
        helperText="Configure choices"
        options={['One', 'Two']}
        onChange={onChange}
      />,
    );

    expect(screen.getByRole('table')).toHaveClass('border-border-subtle');

    fireEvent.change(screen.getByLabelText('suggestion-0'), {
      target: { value: 'First' },
    });
    expect(onChange).toHaveBeenCalledWith(['First', 'Two']);

    fireEvent.click(screen.getAllByLabelText('Remove')[0]);
    expect(onChange).toHaveBeenCalledWith(['Two']);

    fireEvent.click(screen.getByRole('button', { name: 'Add choice' }));
    expect(onChange).toHaveBeenCalledWith(['One', 'Two', '']);

    fireEvent.click(screen.getByRole('button', { name: 'Reverse options' }));
    expect(onChange).toHaveBeenCalledWith(['Two', 'One']);
    expect(screen.getByText('Configure choices')).toBeInTheDocument();
  });

  it('handles prompt editing, role changes, copy feedback, delete, and expand', () => {
    const onChange = jest.fn();
    const onTypeChange = jest.fn();
    const onDelete = jest.fn();

    render(
      <AdvancedPromptInput
        type={PromptRole.system}
        isChatMode
        value="Initial prompt"
        canDelete
        onChange={onChange}
        onTypeChange={onTypeChange}
        onDelete={onDelete}
      />,
    );

    const roleSelector = screen.getByLabelText('Select a role');
    expect(roleSelector.parentElement).toHaveClass('flex-1', 'min-w-0');
    expect(roleSelector.parentElement?.parentElement).toHaveClass(
      'h-10',
      'bg-[var(--cds-field)]',
    );
    expect(roleSelector.parentElement?.nextElementSibling).toHaveClass(
      'flex',
      'shrink-0',
      'items-stretch',
    );

    fireEvent.change(screen.getByLabelText('Select a role'), {
      target: { value: PromptRole.user },
    });
    expect(onTypeChange).toHaveBeenCalledWith(PromptRole.user);

    fireEvent.change(screen.getByLabelText('Prompt editor'), {
      target: { value: 'Updated prompt' },
    });
    expect(onChange).toHaveBeenCalledWith('Updated prompt');

    fireEvent.click(screen.getByRole('button', { name: 'Copy prompt' }));
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      'Initial prompt',
    );
    expect(screen.getByTestId('checkmark-icon')).toBeInTheDocument();
    act(() => {
      jest.advanceTimersByTime(4000);
    });
    expect(screen.getByTestId('copy-icon')).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole('button', { name: 'Delete prompt message' }),
    );
    expect(onDelete).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole('button', { name: 'Expand prompt' }));
    expect(screen.getByTestId('minimize-icon')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Collapse prompt' }),
    ).toBeInTheDocument();
  });
});
