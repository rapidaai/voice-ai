import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { PrimaryButton } from '../button';

describe('PrimaryButton', () => {
  it('uses the shared primary-button and Carbon button-label contracts', () => {
    render(<PrimaryButton size="md">Continue</PrimaryButton>);

    const button = screen.getByRole('button', { name: 'Continue' });
    const label = screen.getByText('Continue');

    expect(button).toHaveClass('cds--btn--primary', 'app-primary-button');
    expect(label).toHaveClass(
      'font-sans',
      'text-[0.875rem]',
      'font-normal',
      'leading-[1.125rem]',
      'tracking-[0.16px]',
      'text-inherit',
    );
  });

  it('preserves caller layout classes on the shared primary button', () => {
    render(
      <PrimaryButton size="lg" className="w-full!">
        Save
      </PrimaryButton>,
    );

    expect(screen.getByRole('button', { name: 'Save' })).toHaveClass(
      'app-primary-button',
      'w-full!',
    );
  });

  it('preserves Carbon disabled and loading states', () => {
    const { rerender } = render(
      <PrimaryButton size="md" disabled>
        Save
      </PrimaryButton>,
    );

    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled();

    rerender(
      <PrimaryButton size="md" isLoading>
        Save
      </PrimaryButton>,
    );

    expect(
      screen.queryByRole('button', { name: 'Save' }),
    ).not.toBeInTheDocument();
    expect(document.querySelector('.cds--skeleton.cds--btn')).toHaveClass(
      'app-primary-button',
      'cds--btn--md',
    );
  });
});
