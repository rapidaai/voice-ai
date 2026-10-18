import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { APiStringHeader } from '@/app/components/domain/external-api/api-header';
import { ParameterEditor } from '@/app/components/domain/tools/common/components';

jest.mock('@/utils', () => ({
  cn: (...inputs: string[]) => inputs.filter(Boolean).join(' '),
}));

jest.mock('@/app/components/ui/primitives/form', () => {
  const React = require('react');
  return {
    TextInput: ({
      id,
      value,
      onChange,
      placeholder,
      labelText,
      hideLabel,
    }: any) =>
      React.createElement(
        'div',
        null,
        !hideLabel && labelText
          ? React.createElement('label', { htmlFor: id }, labelText)
          : null,
        React.createElement('input', {
          id,
          'data-testid': id,
          value: value ?? '',
          placeholder,
          onChange,
        }),
      ),
    TextArea: ({ id, value, onChange }: any) =>
      React.createElement('textarea', { id, value: value ?? '', onChange }),
    Stack: ({ children }: any) => React.createElement('div', null, children),
  };
});

jest.mock('@/app/components/ui/primitives/button', () => {
  const React = require('react');
  return {
    TertiaryButton: ({ children, onClick }: any) =>
      React.createElement('button', { onClick }, children),
  };
});

jest.mock(
  '@/app/components/layout/container/message/notice-block/doc-notice-block',
  () => {
    const React = require('react');
    return {
      DocNoticeBlock: ({ children }: any) =>
        React.createElement('div', null, children),
    };
  },
);

jest.mock('@/app/components/ui/editor/code-editor', () => {
  const React = require('react');
  return {
    CodeEditor: ({ value, onChange }: any) =>
      React.createElement('textarea', {
        'aria-label': 'Code Editor',
        value: value ?? '',
        onChange: (e: any) => onChange(e.target.value),
      }),
  };
});

jest.mock('@carbon/react', () => {
  const React = require('react');
  return {
    Select: ({
      children,
      className,
      hideLabel,
      id,
      labelText,
      onChange,
      value,
    }: any) =>
      React.createElement(
        'div',
        null,
        !hideLabel && labelText
          ? React.createElement('label', { htmlFor: id }, labelText)
          : null,
        React.createElement(
          'select',
          { id, 'data-testid': id, className, value, onChange },
          children,
        ),
      ),
    SelectItem: ({ value, text }: any) =>
      React.createElement('option', { value }, text),
    Button: ({ children, onClick, iconDescription }: any) =>
      React.createElement(
        'button',
        { onClick, 'aria-label': iconDescription || children || 'button' },
        children || 'button',
      ),
    Table: ({ children }: any) => React.createElement('table', null, children),
    TableBody: ({ children }: any) =>
      React.createElement('tbody', null, children),
    TableCell: ({ children, colSpan }: any) =>
      React.createElement('td', { colSpan }, children),
    TableContainer: ({ children }: any) =>
      React.createElement('div', null, children),
    TableHead: ({ children }: any) =>
      React.createElement('thead', null, children),
    TableHeader: ({ children }: any) =>
      React.createElement('th', null, children),
    TableRow: ({ children }: any) => React.createElement('tr', null, children),
  };
});

describe('Tool list editors', () => {
  it('header list supports add/delete and serializes JSON payload', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <APiStringHeader
        headerValue='{"Authorization":"Bearer token"}'
        setHeaderValue={onChange}
      />,
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue('Authorization')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Bearer token')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: 'Add header' }));
    await user.click(screen.getByTestId('api-header-key-1'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('X-Trace'));
    await user.click(screen.getByTestId('api-header-val-1'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('trace-id'));

    const setPayloads = onChange.mock.calls.map(call => JSON.parse(call[0]));
    expect(setPayloads).toContainEqual({
      Authorization: 'Bearer token',
      'X-Trace': 'trace-id',
    });

    await user.click(screen.getAllByRole('button', { name: 'Remove' })[0]);

    const lastCallArg =
      onChange.mock.calls[onChange.mock.calls.length - 1]?.[0] || '{}';
    expect(JSON.parse(lastCallArg)).toEqual({
      'X-Trace': 'trace-id',
    });
  });

  it('header list handles invalid json input fallback', () => {
    const spy = jest.spyOn(console, 'error').mockImplementation(() => {});
    const onChange = jest.fn();

    render(
      <APiStringHeader headerValue="not-json" setHeaderValue={onChange} />,
    );

    expect(screen.getByTestId('api-header-key-0')).toBeInTheDocument();
    expect(screen.getByTestId('api-header-val-0')).toBeInTheDocument();

    spy.mockRestore();
  });

  it('mapping list supports add/edit/delete for tool parameter mapping', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(<ParameterEditor value="{}" onChange={onChange} />);

    expect(screen.getByText('Mapping (0)')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Add parameter' }));
    expect(screen.getByText('Mapping (1)')).toBeInTheDocument();
    expect(screen.getByRole('table')).toHaveClass('border-border-subtle');

    await user.selectOptions(
      screen.getByTestId('param-type-0'),
      String('tool'),
    );
    expect(screen.getByTestId('param-type-0')).toHaveClass('w-full');
    expect(screen.getByTestId('param-type-0').closest('td')).toHaveClass('p-0');
    await user.selectOptions(screen.getByTestId('param-key-0'), String('name'));
    expect(screen.getByTestId('param-key-0')).toHaveClass('w-full');
    await user.click(screen.getByTestId('param-val-0'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('customer_profile'));

    expect(onChange).toHaveBeenLastCalledWith(
      JSON.stringify({ 'tool.name': 'customer_profile' }),
    );

    await user.click(screen.getByRole('button', { name: 'Remove' }));

    expect(onChange).toHaveBeenLastCalledWith('{}');
    expect(screen.getByText('Mapping (0)')).toBeInTheDocument();
  });
});
