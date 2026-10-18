import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { Input } from '../input';
import { InputHelper } from '../input-helper';
import userEvent from '@testing-library/user-event';

jest.mock('@carbon/react', () => {
  const React = require('react');

  return {
    TextInput: React.forwardRef(
      (
        {
          className,
          hideLabel,
          id,
          labelText,
          size,
          ...props
        }: React.InputHTMLAttributes<HTMLInputElement> & {
          hideLabel?: boolean;
          labelText: React.ReactNode;
          size?: string;
        },
        ref: React.ForwardedRef<HTMLInputElement>,
      ) => (
        <label>
          <span className={hideLabel ? 'cds--visually-hidden' : undefined}>
            {labelText}
          </span>
          <input
            {...props}
            ref={ref}
            id={id}
            className={className}
            data-design-system-text-input
            data-size={size}
          />
        </label>
      ),
    ),
  };
});

describe('Input', () => {
  it('renders the Carbon text input with the existing name-based id contract', async () => {
    const user = userEvent.setup();
    const onChange = jest.fn();

    render(
      <Input
        name="workspaceName"
        placeholder="Workspace name"
        className="max-w-80"
        onChange={onChange}
      />,
    );

    const input = screen.getByRole('textbox', { name: 'workspaceName' });

    expect(input).toHaveAttribute('data-design-system-text-input');
    expect(input).toHaveAttribute('id', 'workspaceName');
    expect(input).toHaveAttribute('name', 'workspaceName');
    expect(input).toHaveAttribute('data-size', 'md');
    expect(input).toHaveClass('w-full');
    expect(input).toHaveClass('max-w-80');

    await user.click(input);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support workspace'));

    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it('uses an explicit id and visible label when provided', () => {
    render(
      <Input
        id="organization-email"
        labelText="Organization email"
        hideLabel={false}
        type="email"
      />,
    );

    const input = screen.getByRole('textbox', { name: 'Organization email' });

    expect(input).toHaveAttribute('id', 'organization-email');
    expect(input).toHaveAttribute('type', 'email');
    expect(screen.getByText('Organization email')).toBeVisible();
  });

  it('generates an accessible id when neither id nor name is provided', () => {
    render(<Input aria-label="Document id" disabled value="doc-1" />);

    const input = screen.getByRole('textbox', { name: 'Document id' });

    expect(input.id).toMatch(/^input-/);
    expect(input).toBeDisabled();
    expect(input).toHaveValue('doc-1');
  });
});

describe('InputHelper', () => {
  it('uses the Carbon secondary text token for readable helper copy', () => {
    render(<InputHelper>Helpful context</InputHelper>);

    expect(screen.getByText('Helpful context')).toHaveClass(
      'text-[var(--cds-text-secondary)]',
    );
  });
});
