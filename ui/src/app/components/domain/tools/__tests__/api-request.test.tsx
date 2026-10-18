import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { ConfigureAPIRequest } from '../api-request';

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

const toMap = (items: Metadata[]) =>
  Object.fromEntries(items.map(item => [item.getKey(), item.getValue()]));

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

jest.mock('@/app/components/domain/external-api/api-header', () => ({
  APiStringHeader: () => <div>headers</div>,
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

describe('ConfigureAPIRequest', () => {
  it('supports add and edit mapping parameters', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onParameterChange = jest.fn();

    render(
      <ConfigureAPIRequest
        parameters={[
          createMetadata('tool.method', 'POST'),
          createMetadata('tool.endpoint', 'https://api.example.com'),
          createMetadata('tool.headers', '{}'),
          createMetadata('tool.parameters', '{}'),
        ]}
        onParameterChange={onParameterChange}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Add parameter' }));
    await user.selectOptions(
      screen.getByTestId('param-type-0'),
      String('tool'),
    );
    await user.selectOptions(screen.getByTestId('param-key-0'), String('name'));
    await user.click(screen.getByTestId('param-val-0'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('customer_name'));

    const lastCallArgs = onParameterChange.mock.calls.at(-1)?.[0] as Metadata[];
    const byKey = toMap(lastCallArgs);
    expect(byKey['tool.parameters']).toBe('{"tool.name":"customer_name"}');
  });
});
