import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { SidebarIconWrapper } from '../sidebar-icon-wrapper';
import { SidebarLabel } from '../sidebar-label';
import { SidebarSimpleListItem } from '../sidebar-simple-list-item';
import { typographyClassNames } from '@/app/components/ui/primitives/text';

let mockOpen = true;

jest.mock('@/context/sidebar-context', () => ({
  useSidebar: () => ({ open: mockOpen }),
}));

jest.mock('@carbon/react', () => ({
  SkeletonIcon: ({ className }: { className?: string }) => (
    <span className={className} data-testid="skeleton-icon" />
  ),
  SkeletonText: ({
    className,
    width,
  }: {
    className?: string;
    width: string;
  }) => (
    <span
      className={className}
      data-testid="skeleton-text"
      data-width={width}
    />
  ),
}));

describe('sidebar navigation primitives', () => {
  beforeEach(() => {
    mockOpen = true;
  });

  it('merges icon wrapper classes and forwards attributes', () => {
    render(
      <SidebarIconWrapper className="custom-icon" data-testid="icon-wrapper">
        <svg aria-label="Icon" />
      </SidebarIconWrapper>,
    );

    expect(screen.getByTestId('icon-wrapper')).toHaveClass(
      'custom-icon',
      'h-8',
      'w-12',
      '[&_svg]:w-4',
      '[&_svg]:h-4',
    );
  });

  it('shows labels when the sidebar is open and forwards attributes', () => {
    render(
      <SidebarLabel className="custom-label" data-testid="sidebar-label">
        Dashboard
      </SidebarLabel>,
    );

    expect(screen.getByTestId('sidebar-label')).toHaveClass(
      'custom-label',
      'opacity-100',
    );
    expect(screen.getByText('Dashboard')).toHaveClass(
      ...typographyClassNames['body-compact-01'].split(' '),
    );
  });

  it('collapses labels when the sidebar is closed', () => {
    mockOpen = false;

    render(<SidebarLabel>Dashboard</SidebarLabel>);

    expect(screen.getByText('Dashboard').parentElement).toHaveClass(
      'opacity-0',
      'w-0',
    );
  });

  it('renders active internal links with token classes', () => {
    render(
      <MemoryRouter>
        <SidebarSimpleListItem
          active
          className="custom-item"
          data-testid="sidebar-item"
          navigate="/dashboard"
        >
          Dashboard
        </SidebarSimpleListItem>
      </MemoryRouter>,
    );

    expect(screen.getByRole('link', { name: 'Dashboard' })).toHaveAttribute(
      'href',
      '/dashboard',
    );
    expect(screen.getByTestId('sidebar-item')).toHaveClass(
      'custom-item',
      'h-8',
      'bg-layer-selected',
      'text-foreground',
    );
  });

  it('renders loading placeholders without a link', () => {
    render(
      <MemoryRouter>
        <SidebarSimpleListItem
          className="loading-item"
          loading
          navigate="/dashboard"
        >
          Dashboard
        </SidebarSimpleListItem>
      </MemoryRouter>,
    );

    expect(screen.queryByRole('link')).not.toBeInTheDocument();
    expect(screen.getByTestId('skeleton-icon')).toHaveClass('!h-4', '!w-4');
    expect(screen.getByTestId('skeleton-text')).toHaveAttribute(
      'data-width',
      '70%',
    );
  });
});
