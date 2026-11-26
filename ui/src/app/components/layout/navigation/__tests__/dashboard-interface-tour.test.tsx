import React from 'react';
import { act, renderHook } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { useDashboardInterfaceTour } from '../dashboard-interface-tour';
import { INTERFACE_TOUR_STORAGE_KEY } from '@/context/interface-tour-storage';

let mockAuthId = 'user-1';

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({ authId: mockAuthId }),
}));

function wrapperFor(pathname: string) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <MemoryRouter initialEntries={[pathname]}>{children}</MemoryRouter>;
  };
}

describe('dashboard interface tour', () => {
  beforeEach(() => {
    mockAuthId = 'user-1';
    window.localStorage.clear();
  });

  it('advances through all five interface targets before persisting completion', () => {
    const { result } = renderHook(
      () => useDashboardInterfaceTour({ hasThemeModeAction: true }),
      { wrapper: wrapperFor('/dashboard') },
    );

    expect(result.current.step).toBe('context');

    act(() => result.current.next());
    expect(result.current.step).toBe('theme');

    act(() => result.current.next());
    expect(result.current.step).toBe('account');

    act(() => result.current.next());
    expect(result.current.step).toBe('navigation');

    act(() => result.current.next());
    expect(result.current.step).toBe('sidebar-toggle');

    act(() => result.current.next());
    expect(result.current.step).toBeNull();
    expect(window.localStorage.getItem(INTERFACE_TOUR_STORAGE_KEY)).toBe(
      JSON.stringify({ dashboard: 'completed' }),
    );
  });

  it('stays hidden after dismissal and outside the dashboard', () => {
    window.localStorage.setItem(
      INTERFACE_TOUR_STORAGE_KEY,
      JSON.stringify({ dashboard: 'completed' }),
    );
    const completed = renderHook(
      () => useDashboardInterfaceTour({ hasThemeModeAction: true }),
      { wrapper: wrapperFor('/dashboard') },
    );
    expect(completed.result.current.step).toBeNull();

    window.localStorage.clear();
    const otherRoute = renderHook(
      () => useDashboardInterfaceTour({ hasThemeModeAction: true }),
      { wrapper: wrapperFor('/deployment/assistant') },
    );
    expect(otherRoute.result.current.step).toBeNull();
  });

  it('starts with context and skips theme when mode selection is unavailable', () => {
    const { result } = renderHook(
      () => useDashboardInterfaceTour({ hasThemeModeAction: false }),
      { wrapper: wrapperFor('/dashboard') },
    );

    expect(result.current.step).toBe('context');

    act(() => result.current.next());
    expect(result.current.step).toBe('account');

    act(() => result.current.dismiss());
    expect(result.current.step).toBeNull();
    expect(window.localStorage.getItem(INTERFACE_TOUR_STORAGE_KEY)).toBe(
      JSON.stringify({ dashboard: 'completed' }),
    );
  });

  it('preserves other page progress when completing the dashboard tour', () => {
    window.localStorage.setItem(
      INTERFACE_TOUR_STORAGE_KEY,
      JSON.stringify({ assistants: 'completed' }),
    );
    const { result } = renderHook(
      () => useDashboardInterfaceTour({ hasThemeModeAction: true }),
      { wrapper: wrapperFor('/dashboard') },
    );

    act(() => result.current.dismiss());

    expect(
      JSON.parse(
        window.localStorage.getItem(INTERFACE_TOUR_STORAGE_KEY) ?? '{}',
      ),
    ).toEqual({ assistants: 'completed', dashboard: 'completed' });
  });

  it('reads the existing scalar completion without rewriting it', () => {
    window.localStorage.setItem(INTERFACE_TOUR_STORAGE_KEY, 'completed');
    const { result } = renderHook(
      () => useDashboardInterfaceTour({ hasThemeModeAction: true }),
      { wrapper: wrapperFor('/dashboard') },
    );

    expect(result.current.step).toBeNull();
    expect(window.localStorage.getItem(INTERFACE_TOUR_STORAGE_KEY)).toBe(
      'completed',
    );
  });

  it('does not start before a user identity is available', () => {
    mockAuthId = '';
    const { result } = renderHook(
      () => useDashboardInterfaceTour({ hasThemeModeAction: true }),
      { wrapper: wrapperFor('/dashboard') },
    );

    expect(result.current.step).toBeNull();
  });
});
