import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { Textarea } from '../textarea';
import userEvent from '@testing-library/user-event';

jest.mock('@carbon/react', () => {
  const React = require('react');

  return {
    TextArea: React.forwardRef(
      (
        {
          className,
          hideLabel,
          id,
          labelText,
          ...props
        }: React.TextareaHTMLAttributes<HTMLTextAreaElement> & {
          hideLabel?: boolean;
          labelText: React.ReactNode;
        },
        ref: React.ForwardedRef<HTMLTextAreaElement>,
      ) => (
        <label>
          <span className={hideLabel ? 'cds--visually-hidden' : undefined}>
            {labelText}
          </span>
          <textarea
            {...props}
            ref={ref}
            id={id}
            className={className}
            data-design-system-textarea
          />
        </label>
      ),
    ),
  };
});

describe('Textarea', () => {
  it('renders the Carbon textarea with the existing row alias', async () => {
    const user = userEvent.setup();
    const onChange = jest.fn();

    render(
      <Textarea
        name="description"
        row={4}
        placeholder="Describe the assistant"
        className="max-w-160"
        onChange={onChange}
      />,
    );

    const textarea = screen.getByRole('textbox', { name: 'description' });

    expect(textarea).toHaveAttribute('data-design-system-textarea');
    expect(textarea).toHaveAttribute('id', 'description');
    expect(textarea).toHaveAttribute('name', 'description');
    expect(textarea).toHaveAttribute('rows', '4');
    expect(textarea).toHaveClass('w-full');
    expect(textarea).toHaveClass('max-w-160');

    await user.click(textarea);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Customer support bot'));

    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it('prefers rows over row and shows the label when requested', () => {
    render(
      <Textarea
        id="delete-reason"
        labelText="Delete reason"
        hideLabel={false}
        row={2}
        rows={5}
      />,
    );

    const textarea = screen.getByRole('textbox', { name: 'Delete reason' });

    expect(textarea).toHaveAttribute('id', 'delete-reason');
    expect(textarea).toHaveAttribute('rows', '5');
    expect(screen.getByText('Delete reason')).toBeVisible();
  });

  it('generates an accessible id when neither id nor name is provided', () => {
    render(<Textarea aria-label="Segment reason" disabled value="Outdated" />);

    const textarea = screen.getByRole('textbox', { name: 'Segment reason' });

    expect(textarea.id).toMatch(/^textarea-/);
    expect(textarea).toBeDisabled();
    expect(textarea).toHaveValue('Outdated');
  });
});
