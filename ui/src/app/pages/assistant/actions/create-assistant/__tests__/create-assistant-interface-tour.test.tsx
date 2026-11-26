import { act, renderHook } from '@testing-library/react';
import { INTERFACE_TOUR_STORAGE_KEY } from '@/context/interface-tour-storage';
import { useCreateAssistantInterfaceTour } from '../create-assistant-interface-tour';

describe('create assistant interface tour', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('guides a first visit through template, model, and continue', () => {
    window.localStorage.setItem(
      INTERFACE_TOUR_STORAGE_KEY,
      JSON.stringify({ dashboard: 'completed', assistant: 'completed' }),
    );
    const { result } = renderHook(() =>
      useCreateAssistantInterfaceTour('user-1'),
    );

    expect(result.current.step).toBe('template');

    act(() => result.current.showModel());
    expect(result.current.step).toBe('model');

    act(() => result.current.showContinue());
    expect(result.current.step).toBe('continue');

    act(() => result.current.dismiss());
    expect(result.current.step).toBeNull();
    expect(
      JSON.parse(
        window.localStorage.getItem(INTERFACE_TOUR_STORAGE_KEY) ?? '{}',
      ),
    ).toEqual({
      dashboard: 'completed',
      assistant: 'completed',
      createAssistant: 'completed',
    });
  });

  it('stays hidden after completion and without an authenticated user', () => {
    window.localStorage.setItem(
      INTERFACE_TOUR_STORAGE_KEY,
      JSON.stringify({ createAssistant: 'completed' }),
    );

    const completed = renderHook(() =>
      useCreateAssistantInterfaceTour('user-1'),
    );
    expect(completed.result.current.step).toBeNull();

    window.localStorage.clear();
    const withoutUser = renderHook(() => useCreateAssistantInterfaceTour(''));
    expect(withoutUser.result.current.step).toBeNull();
  });

  it('does not skip ahead when a stale action is invoked', () => {
    const { result } = renderHook(() =>
      useCreateAssistantInterfaceTour('user-1'),
    );

    act(() => result.current.showContinue());
    expect(result.current.step).toBe('template');
  });
});
