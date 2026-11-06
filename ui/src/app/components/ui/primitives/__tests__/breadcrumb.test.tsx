import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { Breadcrumb } from '../breadcrumb';

jest.mock('@carbon/react', () => ({
  Breadcrumb: ({ children }: any) => <nav>{children}</nav>,
  BreadcrumbItem: ({ children, href, 'aria-current': ariaCurrent }: any) => (
    <span data-breadcrumb-item aria-current={ariaCurrent}>
      {href ? <a href={href}>{children}</a> : children}
    </span>
  ),
  BreadcrumbSkeleton: () => <div data-testid="breadcrumb-skeleton" />,
  OverflowMenu: ({ children, iconDescription }: any) => (
    <div data-carbon-breadcrumb-overflow>
      <button type="button" aria-label={iconDescription} />
      {children}
    </div>
  ),
  OverflowMenuItem: ({ href, itemText }: any) => (
    <a data-overflow-breadcrumb-item href={href}>
      {itemText}
    </a>
  ),
}));

describe('Breadcrumb', () => {
  it('moves middle ancestors into the Carbon overflow menu', () => {
    render(
      <Breadcrumb
        maxVisibleItems={3}
        items={[
          { label: 'Home', href: '/' },
          { label: 'Workspace', href: '/workspace' },
          { label: 'Project', href: '/workspace/project' },
          { label: 'Settings', isCurrentPage: true },
        ]}
      />,
    );

    expect(
      screen.getByRole('button', { name: 'Show hidden breadcrumb items' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Workspace')).toHaveAttribute(
      'data-overflow-breadcrumb-item',
    );
    expect(screen.getByText('Workspace')).toHaveAttribute('href', '/workspace');
    expect(screen.getByText('Home').closest('[data-breadcrumb-item]')).not.toBe(
      null,
    );
    expect(
      screen.getByText('Project').closest('[data-breadcrumb-item]'),
    ).not.toBe(null);
    expect(screen.getByText('Settings')).toHaveAttribute(
      'aria-current',
      'page',
    );
  });

  it('does not render overflow when every item fits', () => {
    render(
      <Breadcrumb
        maxVisibleItems={3}
        items={[
          { label: 'Home', href: '/' },
          { label: 'Dashboard', isCurrentPage: true },
        ]}
      />,
    );

    expect(
      screen.queryByRole('button', { name: 'Show hidden breadcrumb items' }),
    ).not.toBeInTheDocument();
    expect(screen.getByText('Dashboard')).toHaveAttribute(
      'aria-current',
      'page',
    );
  });
});
