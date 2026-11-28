import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { DisconnectReasonIndicator } from '../disconnect-reason-indicator';

jest.mock('@carbon/icons-react', () => ({
  Information: () => <svg data-testid="information-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Tag: ({ children }: React.PropsWithChildren) => <span>{children}</span>,
  Toggletip: ({
    align,
    children,
  }: React.PropsWithChildren<{ align: string }>) => (
    <div data-align={align} data-testid="disconnect-toggletip">
      {children}
    </div>
  ),
  ToggletipButton: ({
    children,
    label,
  }: React.PropsWithChildren<{ label: string }>) => (
    <button aria-label={label} type="button">
      {children}
    </button>
  ),
  ToggletipContent: ({ children }: React.PropsWithChildren) => (
    <div>{children}</div>
  ),
}));

describe('DisconnectReasonIndicator', () => {
  it('uses logical Carbon alignment while presenting disconnect details', () => {
    render(
      <DisconnectReasonIndicator reason="OUTBOUND_BUSY" status="FAILED" />,
    );

    expect(screen.getByText('User busy')).toBeVisible();
    expect(screen.getByTestId('disconnect-toggletip')).toHaveAttribute(
      'data-align',
      'bottom-start',
    );
    expect(
      screen.getByRole('button', { name: 'View disconnect details' }),
    ).toBeVisible();
  });
});
