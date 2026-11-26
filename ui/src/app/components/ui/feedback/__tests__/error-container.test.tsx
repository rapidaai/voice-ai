import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { ErrorContainer } from '../error-container';

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => ({ resolvedMode: 'dark' }),
}));

jest.mock('@carbon/icons-react', () => ({
  ArrowLeft: () => <svg data-testid="action-icon" />,
}));

describe('ErrorContainer', () => {
  it('renders a Carbon not-found state for internal 404 errors', () => {
    render(
      <ErrorContainer
        code="404"
        title="Page not found"
        description="The page does not exist."
        actionLabel="Go back"
        onAction={jest.fn()}
      />,
    );

    expect(
      screen.getByRole('heading', { name: 'Page not found' }),
    ).toBeInTheDocument();
    expect(screen.getByText('The page does not exist.')).toBeInTheDocument();
    expect(
      screen
        .getByRole('heading', { name: 'Page not found' })
        .closest('.c4p--empty-state-type--notFound'),
    ).toBeInTheDocument();
    expect(screen.queryByText('404')).not.toBeInTheDocument();
    expect(screen.getByTestId('action-icon')).toBeInTheDocument();
  });

  it('runs the configured action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onAction = jest.fn();

    render(
      <ErrorContainer
        code="500"
        title="Something went wrong"
        description="Try again later."
        actionLabel="Retry"
        onAction={onAction}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Retry' }));

    expect(onAction).toHaveBeenCalledTimes(1);
  });

  it('renders a Carbon unauthorized state for internal 403 errors', () => {
    render(
      <ErrorContainer
        code="403"
        title="Assistant not available"
        description="This assistant may be archived or unavailable."
        actionLabel="Go to listing"
        onAction={jest.fn()}
      />,
    );

    const container = screen.getByLabelText('Assistant not available');
    const title = screen.getByRole('heading', {
      name: 'Assistant not available',
    });

    expect(container).toHaveClass('w-full', 'flex-1', 'items-center');
    expect(title.closest('.c4p--empty-state-type--unauthorized')).toBeVisible();
    expect(screen.queryByText('403')).not.toBeInTheDocument();
    expect(
      screen.getByText('This assistant may be archived or unavailable.'),
    ).toHaveClass('c4p--empty-state__subtitle');
  });
});
