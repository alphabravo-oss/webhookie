import { Store } from '@tanstack/store';
import { act, renderHook } from '@testing-library/react';
import { createStoreHook } from '@/lib/store-hook';

interface CounterState extends Record<string, unknown> {
  count: number;
  label: string;
}

function makeHook(initial: CounterState = { count: 0, label: 'a' }) {
  return createStoreHook(new Store<CounterState>(initial));
}

describe('createStoreHook', () => {
  it('returns the full state when called bare', () => {
    const useCounter = makeHook();
    const { result } = renderHook(() => useCounter());
    expect(result.current).toEqual({ count: 0, label: 'a' });
  });

  it('re-renders with the selected value when the store changes', () => {
    const useCounter = makeHook();
    const { result } = renderHook(() => useCounter((s) => s.count));

    expect(result.current).toBe(0);
    act(() => {
      useCounter.setState({ count: 5 });
    });
    expect(result.current).toBe(5);
  });

  it('getState returns the current state synchronously', () => {
    const useCounter = makeHook();
    expect(useCounter.getState()).toEqual({ count: 0, label: 'a' });
    useCounter.setState({ count: 3 });
    expect(useCounter.getState()).toEqual({ count: 3, label: 'a' });
  });

  describe('setState shallow-merge (legacy persist semantics)', () => {
    it('merges an object partial over the existing state', () => {
      const useCounter = makeHook();
      useCounter.setState({ count: 7 });
      expect(useCounter.getState()).toEqual({ count: 7, label: 'a' });
    });

    it('merges a function partial computed from the previous state', () => {
      const useCounter = makeHook();
      useCounter.setState((prev) => ({ count: prev.count + 1 }));
      useCounter.setState((prev) => ({ count: prev.count + 1 }));
      expect(useCounter.getState()).toEqual({ count: 2, label: 'a' });
    });
  });
});
