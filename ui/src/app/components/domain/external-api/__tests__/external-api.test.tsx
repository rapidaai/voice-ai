import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ApiHeader, ApiStringHeader, APiStringHeader } from '../api-header';
import { ApiParameter, APiParameter } from '../api-parameter';

jest.mock('@carbon/icons-react', () => ({
  Add: () => <svg data-testid="add-icon" />,
  TrashCan: () => <svg data-testid="trash-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Button: ({
    children,
    className: _className,
    hasIconOnly: _hasIconOnly,
    iconDescription,
    kind: _kind,
    onClick,
    renderIcon: Icon,
    size: _size,
    ...props
  }: any) => (
    <button
      aria-label={
        iconDescription || (typeof children === 'string' ? children : undefined)
      }
      onClick={onClick}
      {...props}
    >
      {Icon ? <Icon /> : null}
      {children}
    </button>
  ),
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
}));

jest.mock('@/app/components/ui/primitives/form', () => ({
  TextInput: ({ id, value, onChange, placeholder, labelText }: any) => (
    <label htmlFor={id}>
      {labelText}
      <input
        id={id}
        data-testid={id}
        value={value ?? ''}
        placeholder={placeholder}
        onChange={onChange}
      />
    </label>
  ),
}));

describe('external API key value editors', () => {
  it('edits, adds, and removes header rows', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setHeaders = jest.fn();

    render(
      <ApiHeader
        headers={[
          { key: 'Authorization', value: 'Bearer token' },
          { key: 'X-Trace', value: 'trace-id' },
        ]}
        setHeaders={setHeaders}
      />,
    );

    await user.click(screen.getByTestId('api-header-key-0'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('X-Trace'));
    expect(setHeaders).toHaveBeenCalledWith([
      { key: 'X-Trace', value: 'Bearer token' },
      { key: 'X-Trace', value: 'trace-id' },
    ]);

    await user.click(screen.getByTestId('api-header-val-1'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('trace-id-2'));
    expect(setHeaders).toHaveBeenCalledWith([
      { key: 'Authorization', value: 'Bearer token' },
      { key: 'X-Trace', value: 'trace-id-2' },
    ]);

    await user.click(screen.getByRole('button', { name: 'Add header' }));
    expect(setHeaders).toHaveBeenCalledWith([
      { key: 'Authorization', value: 'Bearer token' },
      { key: 'X-Trace', value: 'trace-id' },
      { key: '', value: '' },
    ]);

    await user.click(screen.getAllByRole('button', { name: 'Remove' })[0]);
    expect(setHeaders).toHaveBeenCalledWith([
      { key: 'X-Trace', value: 'trace-id' },
    ]);
  });

  it('renders key-value controls as a full-cell table form', () => {
    render(
      <ApiHeader
        headers={[{ key: 'Authorization', value: 'Bearer token' }]}
        setHeaders={jest.fn()}
      />,
    );

    expect(screen.getByRole('table').parentElement).toHaveClass(
      '![padding-block-start:0]',
      'border-border-subtle',
    );
    expect(screen.getByTestId('api-header-key-0').closest('td')).toHaveClass(
      '!p-0',
    );
    expect(screen.getByTestId('api-header-val-0').closest('td')).toHaveClass(
      '!p-0',
    );
    expect(screen.getByRole('columnheader', { name: 'Actions' })).toHaveClass(
      '!w-10',
      '!text-center',
    );
    expect(
      screen.getByRole('button', { name: 'Remove' }).closest('td'),
    ).toHaveClass('!w-10', '!p-0', '!text-center');
  });

  it('serializes valid header JSON and ignores blank keys', async () => {
    const user = userEvent.setup();
    const setHeaderValue = jest.fn();

    render(
      <ApiStringHeader
        headerValue='{"Authorization":"Bearer token"}'
        setHeaderValue={setHeaderValue}
      />,
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue('Authorization')).toBeInTheDocument();
    });

    await user.click(screen.getByTestId('api-header-key-0'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('  X-Trace  '));
    expect(JSON.parse(setHeaderValue.mock.calls.at(-1)?.[0])).toEqual({
      'X-Trace': 'Bearer token',
    });

    await user.clear(screen.getByTestId('api-header-key-0'));
    expect(JSON.parse(setHeaderValue.mock.calls.at(-1)?.[0])).toEqual({});
  });

  it('falls back to one empty header row for invalid or non-object JSON', async () => {
    const { rerender } = render(
      <APiStringHeader headerValue="not-json" setHeaderValue={jest.fn()} />,
    );

    await waitFor(() => {
      expect(screen.getByTestId('api-header-key-0')).toHaveValue('');
    });

    rerender(<APiStringHeader headerValue="[]" setHeaderValue={jest.fn()} />);

    await waitFor(() => {
      expect(screen.getByTestId('api-header-val-0')).toHaveValue('');
    });

    rerender(<APiStringHeader headerValue="{}" setHeaderValue={jest.fn()} />);

    await waitFor(() => {
      expect(screen.getByTestId('api-header-key-0')).toHaveValue('');
    });
  });

  it('uses empty JSON when no header value is provided', async () => {
    render(<ApiStringHeader setHeaderValue={jest.fn()} />);

    await waitFor(() => {
      expect(screen.getByTestId('api-header-key-0')).toHaveValue('');
    });
  });

  it('edits, adds, and removes parameter rows with the configured label', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setParameterValue = jest.fn();

    render(
      <ApiParameter
        initialValues={[]}
        setParameterValue={setParameterValue}
        actionButtonLabel="Add parameter"
      />,
    );

    expect(screen.getByText(/No parameter yet/)).toHaveTextContent(
      'No parameter yet. Click Add parameter below.',
    );

    await user.click(screen.getByRole('button', { name: 'Add parameter' }));
    expect(setParameterValue).toHaveBeenCalledWith([{ key: '', value: '' }]);

    render(
      <APiParameter
        initialValues={[{ key: 'body.name', value: 'profile.name' }]}
        setParameterValue={setParameterValue}
      />,
    );

    await user.click(screen.getByTestId('api-param-val-0'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('customer.name'));
    expect(setParameterValue).toHaveBeenCalledWith([
      { key: 'body.name', value: 'customer.name' },
    ]);
  });
});
