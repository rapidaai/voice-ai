import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { SourceConditionRule } from '../source-condition-rule';

const conditionOptions = [
  { label: 'Equals', value: '=' },
  { label: 'Not equals', value: '!=' },
];

const sourceOptions = [
  { label: 'Phone', value: 'phone' },
  { label: 'Web', value: 'web' },
];

const keyOptions = [
  { label: 'Source', value: 'source' },
  { label: 'Mode', value: 'conversation_mode' },
];

const valueOptionsByKey = {
  source: sourceOptions,
  conversation_mode: [
    { label: 'Voice', value: 'voice' },
    { label: 'Text', value: 'text' },
  ],
};

jest.mock('@carbon/icons-react', () => ({
  Add: () => <svg data-testid="add-icon" />,
  Information: () => <svg data-testid="information-icon" />,
  TrashCan: () => <svg data-testid="trash-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Button: ({
    children,
    className,
    disabled,
    hasIconOnly: _hasIconOnly,
    iconDescription,
    kind,
    onClick,
    renderIcon: Icon,
    size: _size,
    ...props
  }: any) => (
    <button
      aria-label={
        iconDescription || (typeof children === 'string' ? children : undefined)
      }
      className={className}
      data-kind={kind}
      disabled={disabled}
      onClick={onClick}
      {...props}
    >
      {Icon ? <Icon /> : null}
      {children}
    </button>
  ),
  Select: ({ children, className, id, labelText, onChange, value }: any) => (
    <label htmlFor={id}>
      {labelText}
      <select
        className={className}
        id={id}
        data-testid={id}
        value={value}
        onChange={onChange}
      >
        {children}
      </select>
    </label>
  ),
  SelectItem: ({ text, value }: any) => <option value={value}>{text}</option>,
  Table: ({ children, className }: any) => (
    <table className={className}>{children}</table>
  ),
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableCell: ({ children, className, colSpan }: any) => (
    <td className={className} colSpan={colSpan}>
      {children}
    </td>
  ),
  TableContainer: ({ children, className }: any) => (
    <section className={className}>{children}</section>
  ),
  TableHead: ({ children }: any) => <thead>{children}</thead>,
  TableHeader: ({ children, className }: any) => (
    <th className={className}>{children}</th>
  ),
  TableRow: ({ children }: any) => <tr>{children}</tr>,
  Tooltip: ({ children, label }: any) => (
    <span title={typeof label === 'string' ? label : undefined}>
      {children}
    </span>
  ),
}));

describe('SourceConditionRule', () => {
  it('renders a default row and adds a new default condition', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeConditions = jest.fn();

    render(
      <SourceConditionRule
        conditions={[]}
        onChangeConditions={onChangeConditions}
        conditionOptions={conditionOptions}
        sourceOptions={sourceOptions}
      />,
    );

    expect(screen.getByText('Rule')).toBeInTheDocument();
    expect(screen.getByTestId('tool-condition-key')).toHaveValue('source');
    expect(screen.getByTestId('tool-condition-op')).toHaveValue('=');
    expect(screen.getByTestId('tool-condition-source-value')).toHaveValue(
      'phone',
    );
    expect(screen.getByRole('button', { name: 'Remove' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Remove' })).toHaveAttribute(
      'data-kind',
      'ghost',
    );
    expect(screen.getByRole('button', { name: 'Remove' })).toHaveClass(
      '!text-[var(--cds-support-error)]',
    );

    await user.click(screen.getByRole('button', { name: 'Add rule' }));

    expect(onChangeConditions).toHaveBeenCalledWith([
      { key: 'source', condition: '=', value: 'phone' },
      { key: 'source', condition: '=', value: 'phone' },
    ]);
  });

  it('updates key, condition, and value while resetting values by key', async () => {
    const user = userEvent.setup();
    const onChangeConditions = jest.fn();

    render(
      <SourceConditionRule
        conditions={[{ key: 'source', condition: '=', value: 'phone' }]}
        onChangeConditions={onChangeConditions}
        conditionOptions={conditionOptions}
        sourceOptions={sourceOptions}
        keyOptions={keyOptions}
        valueOptionsByKey={valueOptionsByKey}
      />,
    );

    await user.selectOptions(
      screen.getByTestId('tool-condition-key'),
      String('conversation_mode'),
    );
    expect(onChangeConditions).toHaveBeenLastCalledWith([
      { key: 'conversation_mode', condition: '=', value: 'voice' },
    ]);

    await user.selectOptions(
      screen.getByTestId('tool-condition-op'),
      String('!='),
    );
    expect(onChangeConditions).toHaveBeenLastCalledWith([
      { key: 'source', condition: '!=', value: 'phone' },
    ]);

    await user.selectOptions(
      screen.getByTestId('tool-condition-source-value'),
      String('web'),
    );
    expect(onChangeConditions).toHaveBeenLastCalledWith([
      { key: 'source', condition: '=', value: 'web' },
    ]);
  });

  it('renders condition controls as a full-cell table form', () => {
    render(
      <SourceConditionRule
        conditions={[{ key: 'source', condition: '=', value: 'phone' }]}
        onChangeConditions={jest.fn()}
        conditionOptions={conditionOptions}
        sourceOptions={sourceOptions}
      />,
    );

    expect(screen.getByRole('table').parentElement).toHaveClass(
      '![padding-block-start:0]',
      'border-border-subtle',
    );
    expect(screen.getByTestId('tool-condition-key')).toHaveClass('w-full');
    expect(screen.getByTestId('tool-condition-key').closest('td')).toHaveClass(
      '!p-0',
    );
    expect(screen.getByTestId('tool-condition-op').closest('td')).toHaveClass(
      '!p-0',
    );
    expect(
      screen.getByTestId('tool-condition-source-value').closest('td'),
    ).toHaveClass('!p-0');
    expect(screen.getByRole('columnheader', { name: 'Actions' })).toHaveClass(
      '!w-10',
      '!text-center',
    );
    expect(
      screen.getByRole('button', { name: 'Remove' }).closest('td'),
    ).toHaveClass('!w-10', '!p-0', '!text-center');
  });

  it('removes rows only when more than one row is present', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeConditions = jest.fn();

    const { rerender } = render(
      <SourceConditionRule
        conditions={[{ key: 'source', condition: '=', value: 'phone' }]}
        onChangeConditions={onChangeConditions}
        conditionOptions={conditionOptions}
        sourceOptions={sourceOptions}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Remove' }));
    expect(onChangeConditions).not.toHaveBeenCalled();

    rerender(
      <SourceConditionRule
        conditions={[
          { key: 'source', condition: '=', value: 'phone' },
          { key: 'source', condition: '!=', value: 'web' },
        ]}
        onChangeConditions={onChangeConditions}
        conditionOptions={conditionOptions}
        sourceOptions={sourceOptions}
      />,
    );

    await user.click(screen.getAllByRole('button', { name: 'Remove' })[0]);

    expect(onChangeConditions).toHaveBeenCalledWith([
      { key: 'source', condition: '!=', value: 'web' },
    ]);
  });

  it('falls back when option lists are empty', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeConditions = jest.fn();

    render(
      <SourceConditionRule
        conditions={[]}
        onChangeConditions={onChangeConditions}
        conditionOptions={[]}
        sourceOptions={[]}
        keyOptions={[]}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Add rule' }));

    expect(onChangeConditions).toHaveBeenCalledWith([
      { key: 'source', condition: '=', value: '' },
      { key: 'source', condition: '=', value: '' },
    ]);
  });

  it('uses an empty value when the selected key has no value options', async () => {
    const user = userEvent.setup();
    const onChangeConditions = jest.fn();

    render(
      <SourceConditionRule
        conditions={[{ key: 'source', condition: '=', value: 'phone' }]}
        onChangeConditions={onChangeConditions}
        conditionOptions={conditionOptions}
        sourceOptions={sourceOptions}
        keyOptions={[
          { label: 'Source', value: 'source' },
          { label: 'Empty key', value: 'empty' },
        ]}
        valueOptionsByKey={{
          empty: [],
        }}
      />,
    );

    await user.selectOptions(
      screen.getByTestId('tool-condition-key'),
      String('empty'),
    );

    expect(onChangeConditions).toHaveBeenCalledWith([
      { key: 'empty', condition: '=', value: '' },
    ]);
  });
});
