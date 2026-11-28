import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { AssistantConversation } from '@rapidaai/react';

import { DurationBreakdownToggletip } from '../duration-breakdown-toggletip';

jest.mock('@carbon/icons-react', () => ({
  Information: () => <svg data-testid="information-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Toggletip: ({
    align,
    children,
  }: React.PropsWithChildren<{ align: string }>) => (
    <div data-align={align} data-testid="duration-toggletip">
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

describe('DurationBreakdownToggletip', () => {
  it('uses logical Carbon alignment while presenting duration details', () => {
    render(
      <DurationBreakdownToggletip
        conversation={new AssistantConversation()}
      />,
    );

    expect(screen.getByTestId('duration-toggletip')).toHaveAttribute(
      'data-align',
      'bottom-start',
    );
    expect(
      screen.getByRole('button', { name: 'View duration breakdown' }),
    ).toBeVisible();
    expect(screen.getByText('conversation.duration_ms')).toBeVisible();
  });
});
