import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { PageHeader, PageSection, PageShell } from '@/components/ui/page';
import { DataTable, type Column } from '@/components/ui/data-table';
import { listFixtures } from '@/lib/api';
import { toastError } from '@/lib/toast';
import type { Fixture } from '@/lib/types';

export const Route = createFileRoute('/fixtures')({
  component: FixturesPage,
});

const columns: Column<Fixture>[] = [
  {
    key: 'provider',
    header: 'Provider',
    width: '180px',
    accessor: (row) => <span className="font-medium">{row.provider}</span>,
    sortAccessor: (row) => row.provider,
  },
  {
    key: 'event',
    header: 'Event',
    width: '180px',
    accessor: (row) => <span className="font-mono text-xs">{row.event}</span>,
    sortAccessor: (row) => row.event,
  },
  {
    key: 'description',
    header: 'Description',
    accessor: (row) => <span className="text-muted-foreground">{row.description}</span>,
    sortAccessor: (row) => row.description,
  },
];

function FixturesPage() {
  const [rows, setRows] = useState<Fixture[]>([]);
  useEffect(() => {
    listFixtures()
      .then((res) => setRows(res.data ?? []))
      .catch((err: Error) => toastError(err.message));
  }, []);
  return (
    <PageShell>
      <PageHeader
        eyebrow="Test"
        title="Fixtures"
        description="Signed sample events webhookie can fire at your receiver."
      />
      <PageSection>
        <DataTable
          data={rows}
          columns={columns}
          keyExtractor={(row) => `${row.provider}:${row.event}`}
          persistKey="fixtures"
          searchPlaceholder="Search fixtures..."
        />
      </PageSection>
    </PageShell>
  );
}
