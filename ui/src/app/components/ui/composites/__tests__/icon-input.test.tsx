import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { SearchIconInput } from '../icon-input';

jest.mock('@/app/components/ui/primitives/button', () => ({
  IconOnlyButton: ({
    className,
    iconDescription,
    kind,
    renderIcon: Icon,
    size,
    type,
  }: any) => (
    <button
      type={type}
      aria-label={iconDescription}
      className={className}
      data-design-system-icon-button-kind={kind}
      data-design-system-icon-button-size={size}
    >
      {Icon ? <Icon /> : null}
    </button>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  Search: ({ className }: any) => (
    <svg data-testid="search-icon" className={className} />
  ),
}));

describe('SearchIconInput', () => {
  it('renders a search input with a Carbon icon submit action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onSubmit = jest.fn((event: React.FormEvent) => {
      event.preventDefault();
    });

    render(
      <form onSubmit={onSubmit}>
        <SearchIconInput
          placeholder="Find knowledge"
          iconClassName="opacity-90"
        />
      </form>,
    );

    const input = screen.getByRole('searchbox', { name: 'Search' });
    expect(input).toHaveAttribute('name', 'search-input');
    expect(input).toHaveAttribute('placeholder', 'Find knowledge');

    const submit = screen.getByRole('button', { name: 'Search' });
    expect(submit).toHaveAttribute(
      'data-design-system-icon-button-kind',
      'ghost',
    );
    expect(submit).toHaveAttribute('data-design-system-icon-button-size', 'sm');
    expect(screen.getByTestId('search-icon')).toHaveClass('opacity-90');

    await user.click(submit);

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it('uses distinct generated ids for repeated search inputs', () => {
    render(
      <>
        <SearchIconInput />
        <SearchIconInput />
      </>,
    );

    const inputs = screen.getAllByRole('searchbox');
    expect(inputs[0]).toHaveAttribute('id');
    expect(inputs[1]).toHaveAttribute('id');
    expect(inputs[0].id).not.toEqual(inputs[1].id);
  });
});
