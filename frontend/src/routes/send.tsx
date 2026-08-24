import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useMemo, useState } from 'react';
import { Send } from 'lucide-react';
import { PageHeader, PageSection, PageShell } from '@/components/ui/page';
import { ActionButton } from '@/components/ui/action-button';
import { DataTable, type Column } from '@/components/ui/data-table';
import { listAttempts, listFixtures, sendEvent } from '@/lib/api';
import { toastError, toastSuccess } from '@/lib/toast';
import type { Fixture, SendAttempt } from '@/lib/types';
import { formatRelativeTime } from '@/lib/utils';

export const Route = createFileRoute('/send')({
  component: SendPage,
});

const attemptCols: Column<SendAttempt>[] = [
  {
    key: 'createdAt',
    header: 'When',
    accessor: (r) => <span className="text-xs text-muted-foreground">{formatRelativeTime(r.createdAt)}</span>,
    sortAccessor: (r) => Date.parse(r.createdAt),
  },
  { key: 'provider', header: 'Provider', accessor: (r) => r.provider, sortAccessor: (r) => r.provider },
  { key: 'eventName', header: 'Event', accessor: (r) => <span className="font-mono text-xs">{r.eventName}</span>, sortAccessor: (r) => r.eventName },
  { key: 'target', header: 'Target', accessor: (r) => <span className="font-mono text-xs">{r.target}</span>, sortAccessor: (r) => r.target },
  {
    key: 'status',
    header: 'Status',
    accessor: (r) => r.error ?? String(r.status ?? '—'),
    sortAccessor: (r) => r.status ?? 0,
  },
  {
    key: 'latencyMs',
    header: 'Latency',
    align: 'right',
    accessor: (r) => <span className="font-mono text-xs">{r.latencyMs}ms</span>,
    sortAccessor: (r) => r.latencyMs,
  },
];

function SendPage() {
  const [fixtures, setFixtures] = useState<Fixture[]>([]);
  const [attempts, setAttempts] = useState<SendAttempt[]>([]);
  const [provider, setProvider] = useState('standard');
  const [event, setEvent] = useState('generic.ping');
  const [target, setTarget] = useState('');
  const [secret, setSecret] = useState('whsec_c2VjcmV0');

  useEffect(() => {
    setTarget(`${window.location.origin}/hooks/generic/default`);
    listFixtures()
      .then((res) => {
        setFixtures(res.data ?? []);
        if (res.data?.[0]) {
          setProvider(res.data[0].provider);
          setEvent(res.data[0].event);
        }
      })
      .catch((err: Error) => toastError(err.message));
    listAttempts()
      .then((res) => setAttempts(res.data ?? []))
      .catch(() => undefined);
  }, []);

  const events = useMemo(() => fixtures.filter((f) => f.provider === provider), [fixtures, provider]);
  const providers = useMemo(() => Array.from(new Set(fixtures.map((f) => f.provider))), [fixtures]);

  return (
    <PageShell>
      <PageHeader
        eyebrow="Test"
        title="Send"
        description="Fire a signed fixture at your app. Standard Webhooks, GitHub, Slack Events, Stripe, and PagerDuty v3."
        actions={
          <ActionButton
            intent="primary"
            icon={<Send className="h-3.5 w-3.5" />}
            onClick={() => {
              sendEvent({ provider, event, target, secret })
                .then((res) => {
                  toastSuccess(`Sent ${event} → ${res.data.status ?? res.data.error}`);
                  return listAttempts();
                })
                .then((res) => setAttempts(res.data ?? []))
                .catch((err: Error) => toastError(err.message));
            }}
          >
            Send fixture
          </ActionButton>
        }
      />
      <PageSection>
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="text-xs">
            Provider
            <select
              className="mt-1 block h-9 w-full rounded-md border border-border bg-background px-2"
              value={provider}
              onChange={(e) => {
                setProvider(e.target.value);
                const first = fixtures.find((f) => f.provider === e.target.value);
                if (first) setEvent(first.event);
              }}
            >
              {providers.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </label>
          <label className="text-xs">
            Event
            <select
              className="mt-1 block h-9 w-full rounded-md border border-border bg-background px-2"
              value={event}
              onChange={(e) => setEvent(e.target.value)}
            >
              {events.map((e) => (
                <option key={e.event} value={e.event}>
                  {e.event}
                </option>
              ))}
            </select>
          </label>
          <label className="text-xs sm:col-span-2">
            Target URL
            <input
              className="mt-1 block h-9 w-full rounded-md border border-border bg-background px-3 font-mono text-xs"
              value={target}
              onChange={(e) => setTarget(e.target.value)}
            />
          </label>
          <label className="text-xs sm:col-span-2">
            Signing secret
            <input
              className="mt-1 block h-9 w-full rounded-md border border-border bg-background px-3 font-mono text-xs"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
            />
          </label>
        </div>
      </PageSection>
      <PageSection title="Attempts">
        <DataTable data={attempts} columns={attemptCols} keyExtractor={(r) => r.id} persistKey="attempts" emptyMessage="No sends yet." />
      </PageSection>
    </PageShell>
  );
}
