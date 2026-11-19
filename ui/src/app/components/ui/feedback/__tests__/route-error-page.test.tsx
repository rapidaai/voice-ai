import React from 'react';
import { render, screen } from '@testing-library/react';

import { RouteErrorPage } from '../route-error-page';

jest.mock('@/app/components/layout/public-shell/public-shell-header', () => ({
  PublicShellHeader: () => <header aria-label="Product header" />,
}));

jest.mock('@/app/components/layout/footer/general-footer', () => ({
  GeneralFooter: () => <footer aria-label="Product footer" />,
}));

jest.mock(
  '@carbon/ibm-products/lib/components/FullPageError/FullPageError',
  () => ({
    FullPageError: ({
      children,
      className,
      description,
      kind,
      label,
      title,
    }: any) => (
      <main className={className} data-kind={kind}>
        <span>{label}</span>
        <h1>{title}</h1>
        <p>{description}</p>
        {children}
      </main>
    ),
  }),
);

describe('RouteErrorPage', () => {
  it('renders Carbon content and a large forwarding link without local styling', () => {
    render(
      <RouteErrorPage
        status={403}
        actionLabel="Go to dashboard"
        actionHref="/dashboard"
      />,
    );

    expect(screen.getByRole('banner')).toHaveAccessibleName('Product header');
    expect(screen.getByRole('contentinfo')).toHaveAccessibleName(
      'Product footer',
    );
    expect(screen.getAllByRole('main')).toHaveLength(1);
    expect(screen.getByRole('main')).not.toHaveAttribute('class');
    expect(screen.getByRole('main')).toHaveAttribute('data-kind', '403');
    expect(screen.getByText('Error 403')).toBeInTheDocument();
    expect(screen.getByText('Access denied')).toBeInTheDocument();
    expect(
      screen.getByText(
        'You are not authorized to access the requested page. Please verify that you are logged in to the hosting environment and your access permissions are correct.',
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: '– Go to dashboard' }),
    ).toHaveAttribute('href', '/dashboard');
  });

  it('renders a custom service-unavailable page without requiring an action', () => {
    render(<RouteErrorPage status={503} />);

    expect(screen.getByText('Error 503')).toBeInTheDocument();
    expect(screen.getByRole('main')).toHaveAttribute('data-kind', 'custom');
    expect(screen.getByText('Service unavailable')).toBeInTheDocument();
    expect(
      screen.getByText(
        'This service is temporarily unavailable. Try again shortly.',
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('renders the default internal-error content', () => {
    render(<RouteErrorPage status={500} />);

    expect(screen.getByText('Error 500')).toBeInTheDocument();
    expect(screen.getByRole('main')).toHaveAttribute('data-kind', 'custom');
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(
      screen.getByText("We couldn't complete your request. Try again."),
    ).toBeInTheDocument();
  });
});
