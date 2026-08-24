import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { Copy } from 'lucide-react';
import { PageHeader, PageSection, PageShell } from '@/components/ui/page';
import { DataTable, type Column } from '@/components/ui/data-table';
import { StatusBadge } from '@/components/ui/status-badge';
import { ActionButton } from '@/components/ui/action-button';
import { CodeBlock } from '@/components/ui/code-block';
import { copyToClipboard } from '@/lib/utils';
import { toastSuccess, toastError } from '@/lib/toast';
import { getMeta, listSinks, patchSink } from '@/lib/api';
import type { Sink } from '@/lib/types';

export const Route = createFileRoute('/sinks')({
  component: SinksPage,
});

function SinksPage() {
  const [sinks, setSinks] = useState<Sink[]>([]);
  const [base, setBase] = useState(typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080');
  const [selected, setSelected] = useState<Sink | null>(null);
  const [delay, setDelay] = useState(0);
  const [status, setStatus] = useState(0);
  const [hang, setHang] = useState(false);

  const reload = () => {
    listSinks()
      .then((res) => setSinks(res.data ?? []))
      .catch((err: Error) => toastError(err.message));
  };

  useEffect(() => {
    reload();
    getMeta()
      .then((res) => {
        if (res.data.publicBaseUrl) setBase(window.location.origin);
      })
      .catch(() => undefined);
  }, []);

  const columns: Column<Sink>[] = [
    {
      key: 'name',
      header: 'Sink',
      accessor: (row) => <span className="font-medium">{row.name}</span>,
      sortAccessor: (row) => row.name,
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
      key: 'path',
      header: 'Path',
      accessor: (row) => <span className="font-mono text-xs text-muted-foreground">{row.path}</span>,
      sortAccessor: (row) => row.path,
    },
    {
      key: 'copy',
      header: '',
      width: '88px',
      align: 'right',
      sortable: false,
      accessor: (row) => (
        <ActionButton
          size="sm"
          intent="ghost"
          icon={<Copy className="h-3.5 w-3.5" />}
          onClick={(e) => {
            e.stopPropagation();
            const url = `${window.location.origin}${row.path}`;
            void copyToClipboard(url).then((ok) => (ok ? toastSuccess('Copied sink URL') : toastError('Failed to copy')));
          }}
        >
          Copy
        </ActionButton>
      ),
    },
  ];

  const slackUrl = `${base}/hooks/slack/services/T00000000/B00000000/webhookie`;

  return (
    <PageShell>
      <PageHeader
        eyebrow="Capture"
        title="Sinks"
        description="Point your app at these URLs instead of Slack, Teams, PagerDuty, or Discord. Paths stay stable across restarts."
      />
      <PageSection>
        <DataTable
          data={sinks}
          columns={columns}
          keyExtractor={(row) => row.id}
          persistKey="sinks"
          searchPlaceholder="Search sinks..."
          onRowClick={(row) => {
            setSelected(row);
            setDelay(row.chaos?.delayMs ?? 0);
            setStatus(row.chaos?.status ?? 0);
            setHang(row.chaos?.hang ?? false);
          }}
        />
      </PageSection>
      {selected ? (
        <PageSection title={`Chaos · ${selected.name}`} description="Delay, override status, or hang to test retries.">
          <div className="flex flex-wrap items-end gap-3">
            <label className="text-xs">
              Delay ms
              <input
                type="number"
                className="mt-1 block h-9 w-28 rounded-md border border-border bg-background px-2"
                value={delay}
                onChange={(e) => setDelay(Number(e.target.value))}
              />
            </label>
            <label className="text-xs">
              Status override
              <input
                type="number"
                className="mt-1 block h-9 w-28 rounded-md border border-border bg-background px-2"
                value={status}
                onChange={(e) => setStatus(Number(e.target.value))}
              />
            </label>
            <label className="flex items-center gap-2 text-xs">
              <input type="checkbox" checked={hang} onChange={(e) => setHang(e.target.checked)} />
              Hang
            </label>
            <ActionButton
              size="sm"
              intent="primary"
              onClick={() => {
                patchSink(selected.id, { delayMs: delay, status, hang, body: '', contentType: '' })
                  .then(() => {
                    toastSuccess('Chaos saved');
                    reload();
                  })
                  .catch((err: Error) => toastError(err.message));
              }}
            >
              Save
            </ActionButton>
          </div>
        </PageSection>
      ) : null}
      <PageSection title="Try it" description="POST a Slack-shaped payload at the seeded incoming webhook.">
        <CodeBlock
          language="bash"
          title="curl"
          code={`curl -s -X POST ${slackUrl} \\\n  -H 'content-type: application/json' \\\n  -d '{"text":"deploy failed","blocks":[{"type":"section","text":{"type":"mrkdwn","text":"*deploy failed*"}}]}'`}
        />
      </PageSection>
    </PageShell>
  );
}
