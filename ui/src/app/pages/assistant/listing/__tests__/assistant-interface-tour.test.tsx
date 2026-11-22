import { act, renderHook } from '@testing-library/react';
import { INTERFACE_TOUR_STORAGE_KEY } from '@/context/interface-tour-storage';
import { useAssistantInterfaceTour } from '../assistant-interface-tour';

describe('assistant interface tour', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('opens the type chooser before persisting assistant completion', () => {
    window.localStorage.setItem(
      INTERFACE_TOUR_STORAGE_KEY,
      JSON.stringify({ dashboard: 'completed' }),
    );
    const { result } = renderHook(() => useAssistantInterfaceTour('user-1'));

    expect(result.current.step).toBe('create');

    act(() => result.current.showTypes());
    expect(result.current.step).toBe('types');

    act(() => result.current.dismiss());
    expect(result.current.step).toBeNull();
    expect(
      JSON.parse(
        window.localStorage.getItem(INTERFACE_TOUR_STORAGE_KEY) ?? '{}',
      ),
    ).toEqual({ dashboard: 'completed', assistant: 'completed' });
  });

  it('stays hidden after assistant completion and without a user', () => {
    window.localStorage.setItem(
      INTERFACE_TOUR_STORAGE_KEY,
      JSON.stringify({ assistant: 'completed' }),
    );

    const completed = renderHook(() => useAssistantInterfaceTour('user-1'));
    expect(completed.result.current.step).toBeNull();

    window.localStorage.clear();
    const withoutUser = renderHook(() => useAssistantInterfaceTour(''));
    expect(withoutUser.result.current.step).toBeNull();
  });

  it('keeps legacy dashboard completion when completing the assistant tour', () => {
    window.localStorage.setItem(INTERFACE_TOUR_STORAGE_KEY, 'completed');
    const { result } = renderHook(() => useAssistantInterfaceTour('user-1'));

    expect(result.current.step).toBe('create');
    act(() => result.current.dismiss());

    expect(
      JSON.parse(
        window.localStorage.getItem(INTERFACE_TOUR_STORAGE_KEY) ?? '{}',
      ),
    ).toEqual({ dashboard: 'completed', assistant: 'completed' });
  });
});
