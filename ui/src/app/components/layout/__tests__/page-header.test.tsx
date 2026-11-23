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
    ).toHaveClass('text-[1.25rem]', 'leading-[1.75rem]', 'font-semibold!');
    expect(screen.getByText('Organization')).toBeInTheDocument();
    expect(screen.getByText('Manage workspace projects.')).toHaveClass(
      'text-muted',
    );
    expect(
      screen.getByRole('button', { name: 'Create project' }),
    ).toBeInTheDocument();
  });

  it('applies an opt-in content boundary to regular and sticky content', () => {
    const { container } = render(
      <PageHeader
        title="Dashboard"
        contentClassName="mx-auto max-w-[1200px]"
      />,
    );

    expect(container.querySelector('[data-page-header-content]')).toHaveClass(
      'mx-auto',
      'max-w-[1200px]',
    );
    expect(
      container.querySelector('[data-page-header-sticky-container]'),
    ).toHaveClass('mx-auto', 'max-w-[1200px]');
  });

  it('supports a compact title without a regular header divider', () => {
    const { container } = render(
      <PageHeader
        title="Dashboard"
        titleVariant="heading-03"
        showDivider={false}
      />,
    );

    expect(
      screen.getByRole('heading', { level: 1, name: 'Dashboard' }),
    ).toHaveClass('text-[1.25rem]', 'leading-[1.75rem]');
    expect(container.querySelector('[data-page-header]')).not.toHaveClass(
      'border-b',
    );
  });

  it('accepts a semantic weight override for the page title', () => {
    render(<PageHeader title="Dashboard" titleClassName="font-semibold!" />);

    expect(
      screen.getByRole('heading', { level: 1, name: 'Dashboard' }),
    ).toHaveClass('font-semibold!');
  });

  it('allows pages to override the shared title and description typography', () => {
    render(
      <PageHeader
        title="Projects"
        description="Manage workspace projects."
        titleVariant="heading-04"
        titleClassName="font-normal!"
        descriptionClassName="text-foreground"
      />,
    );

    expect(
      screen.getByRole('heading', { level: 1, name: 'Projects' }),
    ).toHaveClass('text-[1.75rem]', 'leading-[2.25rem]', 'font-normal!');
    expect(screen.getByText('Manage workspace projects.')).toHaveClass(
      'text-foreground',
    );
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
