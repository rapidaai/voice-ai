import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SidebarProvider, useSidebar } from '@/context/sidebar-context';

function SidebarProbe() {
  const { locked, setLocked } = useSidebar();
  return (
    <button type="button" onClick={() => setLocked(!locked)}>
      {locked ? 'locked' : 'unlocked'}
    </button>
  );
}

describe('SidebarProvider', () => {
  beforeEach(() => {
    window.localStorage.clear();
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('reads and writes the namespaced UI sidenav key', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    window.localStorage.setItem(
      'rpd::__ui_sidenav',
      JSON.stringify({ main: false, assistant: true, endpoint: false }),
    );

    render(
      <SidebarProvider>
        <SidebarProbe />
      </SidebarProvider>,
    );

    expect(screen.getByRole('button')).toHaveTextContent('unlocked');
    await user.click(screen.getByRole('button'));
    expect(
      JSON.parse(window.localStorage.getItem('rpd::__ui_sidenav')!),
    ).toEqual({ main: true, assistant: true, endpoint: false });
  });

  it('reads the existing scalar preference without rewriting it', () => {
    window.localStorage.setItem('rpd::__ui_sidenav', 'false');

    render(
      <SidebarProvider>
        <SidebarProbe />
      </SidebarProvider>,
    );

    expect(screen.getByRole('button')).toHaveTextContent('unlocked');
    expect(window.localStorage.getItem('rpd::__ui_sidenav')).toBe('false');
  });

  it('defaults to a locked sidenav without a stored preference', () => {
    render(
      <SidebarProvider>
        <SidebarProbe />
      </SidebarProvider>,
    );

    expect(screen.getByRole('button')).toHaveTextContent('locked');
  });
});
