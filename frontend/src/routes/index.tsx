import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { Inbox, Radio, AlertTriangle, CheckCircle2 } from 'lucide-react';
import { PageHeader, PageSection, PageShell } from '@/components/ui/page';
import { MetricCard } from '@/components/ui/metric-card';
import { DataTable, type Column } from '@/components/ui/data-table';
import { StatusBadge } from '@/components/ui/status-badge';
import { ActionButton } from '@/components/ui/action-button';
import { Inspector } from '@/components/inspector';
import { formatRelativeTime } from '@/lib/utils';
import { deleteEvents, listEvents } from '@/lib/api';
import { toastError, toastSuccess } from '@/lib/toast';
import type { Event } from '@/lib/types';

export const Route = createFileRoute('/')({
  component: InboxPage,
});

const columns: Column<Event>[] = [
  {
    key: 'receivedAt',
    header: 'Received',
    width: '140px',
    accessor: (row) => (
      <span className="font-mono text-xs text-muted-foreground">{formatRelativeTime(row.receivedAt)}</span>
    ),
    sortAccessor: (row) => Date.parse(row.receivedAt),
  },
  {
    key: 'provider',
    header: 'Provider',
    width: '140px',
    accessor: (row) => <StatusBadge status={row.provider} size="sm" showDot={false} />,
    sortAccessor: (row) => row.provider,
    filter: { label: 'Provider' },
  },
  {
    key: 'method',
    header: 'Method',
    width: '88px',
    accessor: (row) => <span className="font-mono text-xs">{row.method}</span>,
    sortAccessor: (row) => row.method,
  },
  {
    key: 'summary',
    header: 'Summary',
    accessor: (row) => <span className="block max-w-md truncate">{row.summary}</span>,
    sortAccessor: (row) => row.summary,
  },
  {
    key: 'valid',
    header: 'Schema',
    width: '110px',
    accessor: (row) => (
      <StatusBadge status={row.valid ? 'success' : 'failed'} label={row.valid ? 'Valid' : 'Invalid'} size="sm" />
    ),
    sortAccessor: (row) => (row.valid ? 1 : 0),
    filter: { label: 'Schema' },
  },
  {
    key: 'status',
    header: 'Status',
    width: '88px',
    align: 'right',
    accessor: (row) => <span className="font-mono text-xs tabular-nums">{row.status}</span>,
    sortAccessor: (row) => row.status,
  },
  {
    key: 'latencyMs',
    header: 'Latency',
    width: '96px',
    align: 'right',
    accessor: (row) => <span className="font-mono text-xs tabular-nums text-muted-foreground">{row.latencyMs}ms</span>,
    sortAccessor: (row) => row.latencyMs,
  },
];

function InboxPage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    listEvents({ limit: 50 })
      .then((res) => {
        if (!cancelled) setEvents(res.data ?? []);
      })
      .catch((err: Error) => toastError(err.message))
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    const es = new EventSource('/api/v1/events/stream');
    es.addEventListener('webhook', (msg) => {
      const ev = JSON.parse((msg as MessageEvent).data) as Event;
      setEvents((prev) => [ev, ...prev].slice(0, 500));
    });
    es.onerror = () => {
      /* EventSource retries */
    };
    return () => {
      cancelled = true;
      es.close();
    };
  }, []);

  const valid = events.filter((e) => e.valid).length;
  const invalid = events.length - valid;
  const providers = new Set(events.map((e) => e.provider)).size;

  return (
    <PageShell>
      <PageHeader
        eyebrow="Capture"
        title="Inbox"
        description="Live webhook events captured by local sinks. Schema validation, provider previews, and replay land here."
        actions={
          <ActionButton
            size="sm"
            intent="destructive"
            onClick={() => {
              deleteEvents()
                .then(() => {
                  setEvents([]);
                  toastSuccess('Inbox cleared');
                })
                .catch((err: Error) => toastError(err.message));
            }}
          >
            Clear
          </ActionButton>
        }
      />
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard title="Events" value={events.length} subtitle="Loaded" icon={<Inbox className="h-4 w-4" />} />
        <MetricCard title="Valid" value={valid} subtitle="Passed provider schema" icon={<CheckCircle2 className="h-4 w-4" />} />
        <MetricCard title="Invalid" value={invalid} subtitle="Rejected payloads" icon={<AlertTriangle className="h-4 w-4" />} />
        <MetricCard title="Providers" value={providers} subtitle="Seen this session" icon={<Radio className="h-4 w-4" />} />
      </div>
      <PageSection title="Events" description="Search, filter, and open an event to inspect headers, body, and preview.">
        <DataTable
          data={events}
          columns={columns}
          keyExtractor={(row) => row.id}
          persistKey="inbox-events"
          searchPlaceholder="Search events..."
          emptyMessage="No webhooks captured yet. POST to a sink URL to see it here."
          loading={loading}
          onRowClick={(row) => setSelected(row.id)}
          pageSize={20}
        />
      </PageSection>
      {selected ? <Inspector id={selected} onClose={() => setSelected(null)} /> : null}
    </PageShell>
  );
}
