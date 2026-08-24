import { persistedStore } from '@/lib/persisted-store';
import { createStoreHook } from '@/lib/store-hook';

interface UIState extends Record<string, unknown> {
  sidebarCollapsed: boolean;
  commandPaletteOpen: boolean;
  toggleSidebarCollapsed: () => void;
  setCommandPaletteOpen: (open: boolean) => void;
}

export const useUIStore = createStoreHook(
  persistedStore<UIState>(
    {
      sidebarCollapsed: false,
      commandPaletteOpen: false,
      toggleSidebarCollapsed: () =>
        useUIStore.setState((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      setCommandPaletteOpen: (open) => useUIStore.setState({ commandPaletteOpen: open }),
    },
    {
      name: 'webhookie-ui',
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
      }),
    },
  ),
);
