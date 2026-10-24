import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { ErrorContainer } from '../error-container';
import { typographyClassNames } from '@/app/components/ui/primitives/text';

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({ children, renderIcon: Icon, ...props }: any) => (
    <button {...props}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  ArrowLeft: () => <svg data-testid="action-icon" />,
}));

describe('ErrorContainer', () => {
  it('renders the page error content', () => {
    render(
      <ErrorContainer
        code="404"
        title="Page not found"
        description="The page does not exist."
        actionLabel="Go back"
        onAction={jest.fn()}
      />,
    );

    expect(screen.getByText('404')).toBeInTheDocument();
    expect(screen.getByText('Page not found')).toBeInTheDocument();
    expect(screen.getByText('The page does not exist.')).toBeInTheDocument();
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

  it('centers the page state and uses shared Carbon typography', () => {
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
    const code = screen.getByText('403');
    const title = screen.getByRole('heading', {
      name: 'Assistant not available',
    });
    const description = screen.getByText(
      'This assistant may be archived or unavailable.',
    );

    expect(container).toHaveClass('w-full', 'flex-1', 'items-center');
    expect(code).toHaveClass(...typographyClassNames['heading-06'].split(' '));
    expect(title).toHaveClass(
      ...typographyClassNames['heading-04'].split(' '),
    );
    expect(description).toHaveClass(
      ...typographyClassNames['body-01'].split(' '),
    );
    expect(
      screen.getByRole('button', { name: 'Go to listing' }).parentElement,
    ).toHaveClass('mt-8');
  });
});
