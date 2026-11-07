import React from 'react';
import { act, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { PageHeader } from '../page-header';

let intersectionCallback: IntersectionObserverCallback;

describe('PageHeader', () => {
  beforeAll(() => {
    window.IntersectionObserver = class IntersectionObserver {
      readonly root = null;
      readonly rootMargin = '';
      readonly thresholds = [];

      constructor(callback: IntersectionObserverCallback) {
        intersectionCallback = callback;
      }

      observe() {}
      unobserve() {}
      disconnect() {}
      takeRecords() {
        return [];
      }
    };
  });

  it('renders the page context and actions', () => {
    render(
      <PageHeader
        eyebrow="Organization"
        title="Projects"
        description="Manage workspace projects."
        pageActions={<button type="button">Create project</button>}
      />,
    );

    expect(
      screen.getByRole('heading', { level: 1, name: 'Projects' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Organization')).toBeInTheDocument();
    expect(screen.getByText('Manage workspace projects.')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Create project' }),
    ).toBeInTheDocument();
  });

  it('keeps a collapsed title visible on small screens when there are no actions', () => {
    const { container } = render(<PageHeader title="Providers and Models" />);

    act(() => {
      intersectionCallback(
        [{ isIntersecting: false } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      );
    });

    const stickyHeader = container.querySelector(
      '[data-page-header-sticky-content]',
    );
    const stickyTitle = stickyHeader?.querySelector('.text-foreground');

    expect(stickyHeader).toHaveClass('visible', 'opacity-100');
    expect(stickyTitle).toHaveClass('flex');
    expect(stickyTitle).not.toHaveClass('hidden');
    expect(stickyTitle).toHaveTextContent('Providers and Models');
  });
});
