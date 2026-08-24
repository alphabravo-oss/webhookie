import { persistedStore } from '@/lib/persisted-store';

interface TestState extends Record<string, unknown> {
  user: { name: string } | null;
  isAuthenticated: boolean;
  login: (name: string) => void;
}

function makeInitial(): TestState {
  const initial: TestState = {
    user: null,
    isAuthenticated: false,
    login: () => {},
  };
  return initial;
}

describe('persistedStore', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  describe('envelope byte-compat', () => {
    it('writes the exact legacy persist envelope {"state":…,"version":N}', () => {
      const store = persistedStore(makeInitial(), {
        name: 'astronomer-test',
        version: 2,
        partialize: (s) => ({ user: s.user, isAuthenticated: s.isAuthenticated }),
      });

      store.setState((prev) => ({ ...prev, user: { name: 'ada' }, isAuthenticated: true }));

      expect(localStorage.getItem('astronomer-test')).toBe(
        '{"state":{"user":{"name":"ada"},"isAuthenticated":true},"version":2}',
      );
    });

    it('defaults version to 0 like the legacy persist middleware', () => {
      const store = persistedStore(makeInitial(), {
        name: 'astronomer-test',
        partialize: (s) => ({ isAuthenticated: s.isAuthenticated }),
      });

      store.setState((prev) => ({ ...prev, isAuthenticated: true }));

      expect(localStorage.getItem('astronomer-test')).toBe(
        '{"state":{"isAuthenticated":true},"version":0}',
      );
    });
  });

  describe('hydration', () => {
    it('spreads persisted state over the initial state, keeping functions', () => {
      localStorage.setItem(
        'astronomer-test',
        JSON.stringify({ state: { isAuthenticated: true }, version: 2 }),
      );
      const initial = makeInitial();

      const store = persistedStore(initial, { name: 'astronomer-test', version: 2 });

      expect(store.state.isAuthenticated).toBe(true);
      expect(store.state.user).toBeNull();
      expect(store.state.login).toBe(initial.login);
    });

    it('runs migrate on version mismatch', () => {
      localStorage.setItem(
        'astronomer-test',
        JSON.stringify({
          state: { user: { name: 'ada' }, isAuthenticated: true, token: 'legacy' },
          version: 1,
        }),
      );
      const migrate = vi.fn((persisted: unknown) => {
        const { token: _token, ...rest } = persisted as Record<string, unknown>;
        return rest;
      });

      const store = persistedStore(makeInitial(), {
        name: 'astronomer-test',
        version: 2,
        migrate,
      });

      expect(migrate).toHaveBeenCalledWith(expect.objectContaining({ token: 'legacy' }), 1);
      expect(store.state.user).toEqual({ name: 'ada' });
      expect(store.state.isAuthenticated).toBe(true);
      expect('token' in store.state).toBe(false);
    });

    it('does not run migrate when versions match', () => {
      localStorage.setItem(
        'astronomer-test',
        JSON.stringify({ state: { isAuthenticated: true }, version: 2 }),
      );
      const migrate = vi.fn();

      const store = persistedStore(makeInitial(), {
        name: 'astronomer-test',
        version: 2,
        migrate,
      });

      expect(migrate).not.toHaveBeenCalled();
      expect(store.state.isAuthenticated).toBe(true);
    });

    it('discards persisted state on version mismatch without migrate (legacy parity)', () => {
      localStorage.setItem(
        'astronomer-test',
        JSON.stringify({ state: { isAuthenticated: true }, version: 1 }),
      );

      const store = persistedStore(makeInitial(), { name: 'astronomer-test', version: 2 });

      expect(store.state.isAuthenticated).toBe(false);
    });

    it('falls back to the initial state on corrupted JSON', () => {
      localStorage.setItem('astronomer-test', '{not json!');

      const initial = makeInitial();
      const store = persistedStore(initial, { name: 'astronomer-test', version: 2 });

      expect(store.state).toBe(initial);
      // Writes still work afterwards.
      store.setState((prev) => ({ ...prev, isAuthenticated: true }));
      expect(JSON.parse(localStorage.getItem('astronomer-test')!)).toEqual({
        state: expect.objectContaining({ isAuthenticated: true }),
        version: 2,
      });
    });
  });

  describe('persistence', () => {
    it('partialize excludes functions from the persisted envelope', () => {
      const store = persistedStore(makeInitial(), {
        name: 'astronomer-test',
        partialize: (s) => ({ user: s.user, isAuthenticated: s.isAuthenticated }),
      });

      store.setState((prev) => ({ ...prev, isAuthenticated: true }));

      const envelope = JSON.parse(localStorage.getItem('astronomer-test')!);
      expect(Object.keys(envelope.state).sort()).toEqual(['isAuthenticated', 'user']);
      expect(envelope.state).not.toHaveProperty('login');
    });

    it('stays memory-only when localStorage writes throw (quota errors)', () => {
      const store = persistedStore(makeInitial(), { name: 'astronomer-test' });
      const setItem = vi
        .spyOn(Storage.prototype, 'setItem')
        .mockImplementation(() => {
          throw new DOMException('quota exceeded', 'QuotaExceededError');
        });

      try {
        expect(() =>
          store.setState((prev) => ({ ...prev, isAuthenticated: true })),
        ).not.toThrow();
        expect(store.state.isAuthenticated).toBe(true);
      } finally {
        setItem.mockRestore();
      }
    });
  });
});
