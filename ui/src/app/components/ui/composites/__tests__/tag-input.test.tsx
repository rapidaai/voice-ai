import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { TagInput } from '../tag-input';

jest.mock('@carbon/icons-react', () => ({
  Information: () => <svg data-testid="information-icon" />,
}));

jest.mock('@carbon/react', () => {
  const { forwardRef } = jest.requireActual('react') as typeof import('react');

  return {
    DismissibleTag: ({ onClose, text }: any) => (
      <button type="button" onClick={onClose}>
        {text}
      </button>
    ),
    Tag: ({ children, onClick }: React.PropsWithChildren<any>) => (
      <button type="button" onClick={onClick}>
        {children}
      </button>
    ),
    Toggletip: ({ children }: React.PropsWithChildren) => <>{children}</>,
    ToggletipButton: ({ children, label }: React.PropsWithChildren<any>) => (
      <button type="button" aria-label={label}>
        {children}
      </button>
    ),
    ToggletipContent: ({ children }: React.PropsWithChildren) => (
      <div>{children}</div>
    ),
    TextInput: forwardRef<HTMLInputElement, any>(
      (
        { hideLabel, id, labelText, onChange, onKeyDown, placeholder, value },
        ref,
      ) =>
        hideLabel ? (
          <input
            aria-label={labelText}
            id={id}
            onChange={onChange}
            onKeyDown={onKeyDown}
            placeholder={placeholder}
            ref={ref}
            value={value}
          />
        ) : (
          <label htmlFor={id}>
            {labelText}
            <input
              id={id}
              onChange={onChange}
              onKeyDown={onKeyDown}
              placeholder={placeholder}
              ref={ref}
              value={value}
            />
          </label>
        ),
    ),
  };
});

describe('TagInput', () => {
  it('adds a typed tag on Enter and clears the field', async () => {
    const user = userEvent.setup();
    const addTag = jest.fn();

    render(
      <TagInput tags={[]} addTag={addTag} removeTag={jest.fn()} allTags={[]} />,
    );

    const input = screen.getByRole('textbox', { name: 'Tags (Optional)' });
    await user.click(input);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('priority'));
    await user.type(input, '{Enter}');

    expect(addTag).toHaveBeenCalledWith('priority');
    expect(input).toHaveValue('');
  });

  it('does not add empty values and removes existing tags', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const addTag = jest.fn();
    const removeTag = jest.fn();

    render(
      <TagInput
        tags={['production']}
        addTag={addTag}
        removeTag={removeTag}
        allTags={['production']}
      />,
    );

    await user.type(screen.getByRole('textbox'), '{Enter}');
    await user.click(screen.getByRole('button', { name: 'production' }));

    expect(addTag).not.toHaveBeenCalled();
    expect(removeTag).toHaveBeenCalledWith('production');
  });

  it('offers only tags that are not already selected', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const addTag = jest.fn();

    render(
      <TagInput
        tags={['billing']}
        addTag={addTag}
        removeTag={jest.fn()}
        allTags={['billing', 'support']}
      />,
    );

    expect(screen.getAllByRole('button', { name: 'billing' })).toHaveLength(1);

    await user.click(screen.getByRole('button', { name: 'support' }));

    expect(addTag).toHaveBeenCalledWith('support');
  });
});
