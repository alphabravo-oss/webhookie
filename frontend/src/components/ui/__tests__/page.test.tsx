import { render, screen } from '@testing-library/react';
import { PageHeader, PageSection, PageShell } from '@/components/ui/page';

describe('Page layout primitives', () => {
  it('renders a page shell with standard spacing', () => {
    const { container } = render(
      <PageShell>
        <div>Content</div>
      </PageShell>
    );

    expect(screen.getByText('Content')).toBeInTheDocument();
    expect(container.firstChild).toHaveClass('space-y-6');
  });

  it('renders title, description, eyebrow, and actions', () => {
    render(
      <PageHeader
        eyebrow="Operations"
        title="Backups"
        description="Restore points and storage targets."
        actions={<button>Create</button>}
      />
    );

    expect(screen.getByText('Operations')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Backups' })).toBeInTheDocument();
    expect(screen.getByText('Restore points and storage targets.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create' })).toBeInTheDocument();
  });

  it('renders an unframed page section with optional heading actions', () => {
    render(
      <PageSection
        title="Instances"
        description="Registered control planes."
        actions={<button>Refresh</button>}
      >
        <div>Table</div>
      </PageSection>
    );

    expect(screen.getByRole('heading', { name: 'Instances' })).toBeInTheDocument();
    expect(screen.getByText('Registered control planes.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument();
    expect(screen.getByText('Table')).toBeInTheDocument();
  });
});
