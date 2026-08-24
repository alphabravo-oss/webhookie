import { Store } from '@tanstack/store';

export interface PersistedStoreOptions<T> {
  /** localStorage key. */
  name: string;
  /** Envelope version; defaults to 0 (legacy persist-middleware default). */
  version?: number;
  /** Runs when the persisted envelope version differs from `version`. */
  migrate?: (persistedState: unknown, version: number) => unknown;
  /** Subset of state to persist; defaults to the whole state. */
  partialize?: (state: T) => Record<string, unknown>;
}

/**
 * TanStack Store wrapped with legacy persist-middleware envelope semantics (D21):
 * hydrates from `{"state":…,"version":N}`, runs `migrate` on version
 * mismatch, spreads persisted state over the initial state, and writes the
 * `partialize`d state back on every update. Byte-compatible with the
 * pre-migration persisted envelopes.
 */
export function persistedStore<T extends Record<string, unknown>>(
  initial: T,
  options: PersistedStoreOptions<T>,
): Store<T> {
  const { name, version = 0, migrate, partialize } = options;

  let seed = initial;
  try {
    const raw = localStorage.getItem(name);
    if (raw) {
      const envelope = JSON.parse(raw) as { state?: unknown; version?: number };
      let state = envelope.state;
      if (envelope.version !== version) {
        // Zustand parity: a version mismatch without a migrate function
        // discards the persisted state.
        state = migrate ? migrate(state, envelope.version ?? 0) : undefined;
      }
      if (state && typeof state === 'object') {
        seed = { ...initial, ...(state as Partial<T>) };
      }
    }
  } catch {
    // Corrupted JSON or unavailable storage: start from the initial state.
  }

  const store = new Store<T>(seed);
  store.subscribe(() => {
    try {
      const state = partialize ? partialize(store.state) : store.state;
      localStorage.setItem(name, JSON.stringify({ state, version }));
    } catch {
      // Quota exceeded or unavailable storage: keep working memory-only.
    }
  });
  return store;
}
