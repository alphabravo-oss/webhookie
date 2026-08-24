import { render, screen, act } from '@testing-library/react';
import { ThemeProvider, useTheme, THEME_STORAGE_KEY } from './theme';

// jsdom has no matchMedia; stub one with controllable matches + listeners so
// the provider's system tracking can be exercised.
let systemPrefersDark = false;
const mqListeners: Array<(e: { matches: boolean }) => void> = [];

function flipSystemPreference(matches: boolean) {
  systemPrefersDark = matches;
  act(() => {
    for (const listener of [...mqListeners]) listener({ matches });
  });
}

function Probe() {
  const { theme, setTheme } = useTheme();
  return (
    <>
      <span data-testid="theme">{theme}</span>
      <button onClick={() => setTheme('light')}>go-light</button>
    </>
  );
}

describe('ThemeProvider', () => {
  beforeEach(() => {
    systemPrefersDark = false;
    mqListeners.length = 0;
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      get matches() {
        return systemPrefersDark;
      },
      media: query,
      addEventListener: (_: string, cb: (e: { matches: boolean }) => void) => {
        mqListeners.push(cb);
      },
      removeEventListener: (_: string, cb: (e: { matches: boolean }) => void) => {
        const i = mqListeners.indexOf(cb);
        if (i !== -1) mqListeners.splice(i, 1);
      },
    })) as unknown as typeof window.matchMedia;
    localStorage.clear();
    document.documentElement.classList.remove('dark');
    document.documentElement.style.colorScheme = '';
  });

  it('uses the literal webhookie-theme storage key (never bare `theme`)', () => {
    expect(THEME_STORAGE_KEY).toBe('webhookie-theme');
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>
    );
    act(() => screen.getByText('go-light').click());
    expect(localStorage.getItem('webhookie-theme')).toBe('light');
    expect(localStorage.getItem('theme')).toBeNull();
  });

  it('defaults to dark when nothing is stored', () => {
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>
    );
    expect(screen.getByTestId('theme')).toHaveTextContent('dark');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe('dark');
  });

  it('initializes from the stored raw value', () => {
    localStorage.setItem('webhookie-theme', 'light');
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>
    );
    expect(screen.getByTestId('theme')).toHaveTextContent('light');
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    expect(document.documentElement.style.colorScheme).toBe('light');
  });

  it('tracks system preference changes while theme is system', () => {
    localStorage.setItem('webhookie-theme', 'system');
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>
    );
    expect(screen.getByTestId('theme')).toHaveTextContent('system');
    expect(document.documentElement.classList.contains('dark')).toBe(false);

    flipSystemPreference(true);
    expect(document.documentElement.classList.contains('dark')).toBe(true);

    flipSystemPreference(false);
    expect(document.documentElement.classList.contains('dark')).toBe(false);
  });
});
