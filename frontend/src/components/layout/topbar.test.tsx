import { fireEvent, render, screen } from '@testing-library/react';
import { Topbar } from './topbar';
import { ThemeProvider } from '@/lib/theme';

vi.mock('@/lib/navigation', () => ({
  usePathname: () => '/sinks',
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn() }),
}));

describe('Topbar chrome', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove('dark');
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })) as unknown as typeof window.matchMedia;
  });

  it('renders breadcrumbs and theme + command controls', () => {
    render(
      <ThemeProvider>
        <Topbar />
      </ThemeProvider>,
    );
    expect(screen.getByText('Inbox')).toBeInTheDocument();
    expect(screen.getByText('Sinks')).toBeInTheDocument();
    expect(screen.getByText('K')).toBeInTheDocument();
    expect(screen.getByLabelText(/Theme:/)).toBeInTheDocument();
  });

  it('cycles theme dark → system → light (Astronomer order)', () => {
    render(
      <ThemeProvider>
        <Topbar />
      </ThemeProvider>,
    );
    const button = screen.getByLabelText(/Theme:/);
    fireEvent.click(button);
    expect(localStorage.getItem('webhookie-theme')).toBe('system');
    fireEvent.click(button);
    expect(localStorage.getItem('webhookie-theme')).toBe('light');
    fireEvent.click(button);
    expect(localStorage.getItem('webhookie-theme')).toBe('dark');
  });
});
