import type { ComponentProps } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { Lock, Plus } from 'lucide-react';
import { EmptyState, ErrorState, LoadingState, PermissionState } from '@/components/ui/empty-state';

// Plain-anchor stand-in: these tests assert link text/href, not routing, and
// the real Link needs a <RouterProvider>.
vi.mock('@/lib/link', () => ({
  Link: ({ href, children, ...rest }: ComponentProps<'a'>) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));

describe('EmptyState', () => {
  it('renders title and description', () => {
    render(
      <EmptyState
        icon={Lock}
        title="Admins only"
        description="This surface is gated to platform administrators."
      />,
    );

    expect(screen.getByText('Admins only')).toBeInTheDocument();
    expect(screen.getByText('This surface is gated to platform administrators.')).toBeInTheDocument();
  });

  it('renders an action button and invokes the handler', async () => {
    const onAction = vi.fn();

    render(
      <EmptyState
        icon={Plus}
        title="No storage locations yet"
        description="Add a storage target before scheduling backups."
        actionLabel="Add Storage"
        actionIcon={Plus}
        onAction={onAction}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /add storage/i }));

    expect(onAction).toHaveBeenCalledTimes(1);
  });

  it('disables action buttons', () => {
    const onAction = vi.fn();

    render(
      <EmptyState
        icon={Plus}
        title="No schedules configured"
        description="Create a storage location first."
        actionLabel="Create Schedule"
        actionIcon={Plus}
        onAction={onAction}
        disabled
      />,
    );

    expect(screen.getByRole('button', { name: /create schedule/i })).toBeDisabled();
  });

  it('renders action links', () => {
    render(
      <EmptyState
        icon={Lock}
        title="Section disabled"
        description="This section is disabled by platform settings."
        actionLabel="Back to dashboard"
        actionHref="/dashboard"
      />,
    );

    expect(screen.getByRole('link', { name: /back to dashboard/i })).toHaveAttribute('href', '/dashboard');
  });

  it('renders a shared loading state', () => {
    render(<LoadingState title="Loading quota usage" description="Fetching current consumption." />);

    expect(screen.getByText('Loading quota usage')).toBeInTheDocument();
    expect(screen.getByText('Fetching current consumption.')).toBeInTheDocument();
  });

  it('renders retryable shared errors', () => {
    const onRetry = vi.fn();

    render(<ErrorState title="Load failed" description="The API returned an error." onRetry={onRetry} />);

    fireEvent.click(screen.getByRole('button', { name: /retry/i }));

    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('renders permission-specific copy', () => {
    render(<PermissionState permission="cluster_templates:read" />);

    expect(screen.getByText('Permission required')).toBeInTheDocument();
    expect(screen.getByText('cluster_templates:read')).toBeInTheDocument();
  });
});
