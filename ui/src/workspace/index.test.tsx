import { renderHook } from '@testing-library/react';
import { CONFIG } from '@/configs';
import { useWorkspace } from '@/workspace';

describe('useWorkspace', () => {
  it('returns the configured workspace', () => {
    const { result } = renderHook(() => useWorkspace());

    expect(result.current).toBe(CONFIG.workspace);
  });

  it('keeps the workspace reference stable across renders', () => {
    const { result, rerender } = renderHook(() => useWorkspace());
    const initialWorkspace = result.current;

    rerender();

    expect(result.current).toBe(initialWorkspace);
  });
});
