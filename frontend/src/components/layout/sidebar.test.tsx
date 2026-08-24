import type { ComponentProps } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { Sidebar } from './sidebar';

vi.mock('@/lib/link', () => ({
  Link: ({ href, children, ...rest }: ComponentProps<'a'>) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));

vi.mock('@/lib/navigation', () => ({
  usePathname: () => '/',
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn() }),
}));

describe('Sidebar chrome', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('renders the Webhookie wordmark and AlphaBravo lockup', () => {
    render(<Sidebar />);
    expect(screen.getByText('Webhookie')).toBeInTheDocument();
    expect(screen.getByText('by AlphaBravo')).toBeInTheDocument();
    expect(screen.getByText('Inbox')).toBeInTheDocument();
    expect(screen.getByText('Sinks')).toBeInTheDocument();
    expect(screen.getByText('Send')).toBeInTheDocument();
    expect(screen.getByText('Fixtures')).toBeInTheDocument();
    expect(screen.getByText('Slack')).toBeInTheDocument();
    expect(screen.getByText('Teams')).toBeInTheDocument();
    expect(screen.getByText('Discord')).toBeInTheDocument();
    expect(screen.getByText('PagerDuty')).toBeInTheDocument();
    expect(screen.getByText('Telegram')).toBeInTheDocument();
    expect(screen.getByText('Google Chat')).toBeInTheDocument();
    expect(screen.getByText('Mattermost')).toBeInTheDocument();
    expect(screen.getByText('Opsgenie')).toBeInTheDocument();
  });

  it('collapses to icon rail', () => {
    render(<Sidebar />);
    fireEvent.click(screen.getByLabelText('Collapse sidebar'));
    expect(screen.queryByText('Webhookie')).not.toBeInTheDocument();
    expect(screen.getByLabelText('Expand sidebar')).toBeInTheDocument();
  });
});
