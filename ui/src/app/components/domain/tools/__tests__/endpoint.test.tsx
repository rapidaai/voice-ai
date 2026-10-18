import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { ConfigureEndpoint } from '../endpoint';

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

const toMap = (items: Metadata[]) =>
  Object.fromEntries(items.map(item => [item.getKey(), item.getValue()]));

jest.mock('@/utils', () => ({
  cn: (...inputs: string[]) => inputs.filter(Boolean).join(' '),
}));

jest.mock('@/app/components/ui/primitives/input-group', () => ({
  InputGroup: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/ui/primitives/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: ({ id, value, onChange, labelText, hideLabel }: any) => (
    <div>
      {!hideLabel && labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <input id={id} data-testid={id} value={value ?? ''} onChange={onChange} />
    </div>
  ),
  TextArea: ({ id, value, onChange }: any) => (
    <textarea id={id} value={value ?? ''} onChange={onChange} />
  ),
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  TertiaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@carbon/react', () => ({
  Select: ({ id, value, onChange, children, hideLabel, labelText }: any) => (
    <div>
      {!hideLabel && labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <select id={id} data-testid={id} value={value} onChange={onChange}>
        {children}
      </select>
    </div>
  ),
  SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
  Button: ({ children, iconDescription, ...props }: any) => (
    <button aria-label={iconDescription || children || 'button'} {...props}>
      {children || 'button'}
    </button>
  ),
  Tooltip: ({ children }: any) => <span>{children}</span>,
}));

jest.mock('@/app/components/domain/dropdowns/endpoint-dropdown', () => ({
  EndpointDropdown: ({ className, onChangeEndpoint }: any) => (
    <button
      type="button"
      className={className}
      onClick={() =>
        onChangeEndpoint({
          getId: () => 'endpoint-1',
        })
      }
    >
      Pick endpoint
    </button>
  ),
}));

jest.mock(
  '@/app/components/layout/container/message/notice-block/doc-notice-block',
  () => ({
    DocNoticeBlock: ({ children }: any) => <div>{children}</div>,
  }),
);

jest.mock('@/app/components/ui/editor/code-editor', () => ({
  CodeEditor: ({ value, onChange }: any) => (
    <textarea value={value ?? ''} onChange={e => onChange(e.target.value)} />
  ),
}));

describe('ConfigureEndpoint', () => {
  it('supports add and edit mapping parameters', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onParameterChange = jest.fn();

    render(
      <ConfigureEndpoint
        parameters={[
          createMetadata('tool.endpoint_id', ''),
          createMetadata('tool.parameters', '{}'),
        ]}
        onParameterChange={onParameterChange}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Pick endpoint' }));
    await user.click(screen.getByRole('button', { name: 'Add parameter' }));
    await user.selectOptions(
      screen.getByTestId('param-type-0'),
      String('tool'),
    );
    await user.selectOptions(
      screen.getByTestId('param-key-0'),
      String('argument'),
    );
    await user.click(screen.getByTestId('param-val-0'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('customer_id'));

    const endpointIdCalls = onParameterChange.mock.calls.map(
      call => toMap(call[0] as Metadata[])['tool.endpoint_id'],
    );
    const parameterCalls = onParameterChange.mock.calls.map(
      call => toMap(call[0] as Metadata[])['tool.parameters'],
    );

    expect(endpointIdCalls).toContain('endpoint-1');
    expect(parameterCalls).toContain('{"tool.argument":"customer_id"}');
  });

  it('does not force a light wrapper around the endpoint dropdown', () => {
    render(
      <ConfigureEndpoint
        parameters={[
          createMetadata('tool.endpoint_id', ''),
          createMetadata('tool.parameters', '{}'),
        ]}
        onParameterChange={jest.fn()}
        inputClass="domain-input"
      />,
    );

    expect(screen.getByRole('button', { name: 'Pick endpoint' })).toHaveClass(
      'domain-input',
    );
    expect(
      screen.getByRole('button', { name: 'Pick endpoint' }),
    ).not.toHaveClass('bg-light-background');
  });
});
