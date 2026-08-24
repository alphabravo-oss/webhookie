import { useStore } from '@tanstack/react-store';
import type { Store } from '@tanstack/store';

/**
 * Wraps a TanStack Store in a legacy-shaped hook: callable bare or with a
 * selector, plus `getState()` and shallow-merging `setState(partial | fn)`
 * (imperative test/call sites rely on partial setState).
 */
export function createStoreHook<T extends Record<string, unknown>>(store: Store<T>) {
  function useBoundStore(): T;
  function useBoundStore<U>(selector: (state: T) => U): U;
  function useBoundStore<U>(selector?: (state: T) => U): T | U {
    return useStore(store, selector as (state: T) => U);
  }
  useBoundStore.getState = (): T => store.state;
  useBoundStore.setState = (partial: Partial<T> | ((state: T) => Partial<T>)): void => {
    store.setState((prev) => ({
      ...prev,
      ...(typeof partial === 'function' ? partial(prev) : partial),
    }));
  };
  return useBoundStore;
}
