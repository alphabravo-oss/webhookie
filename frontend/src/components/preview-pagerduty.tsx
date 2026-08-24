import { StatusBadge } from '@/components/ui/status-badge';
import type { Event } from '@/lib/types';
import { formatRelativeTime } from '@/lib/utils';

export function PagerDutyPreview({ body, related }: { body: string; related: Event[] }) {
  let payload: { event_action?: string; dedup_key?: string; payload?: { summary?: string; severity?: string } } = {};
  try {
    payload = JSON.parse(body);
  } catch {
    return <p className="text-sm text-muted-foreground">Not JSON</p>;
  }
  const items = related.length ? related : [];
  return (
    <div className="space-y-3">
      <div className="rounded-lg border border-border bg-card p-4">
        <p className="text-sm font-medium">{payload.payload?.summary ?? payload.event_action ?? 'PagerDuty event'}</p>
        <div className="mt-2 flex gap-2">
          {payload.event_action ? <StatusBadge status={payload.event_action} size="sm" /> : null}
          {payload.payload?.severity ? <StatusBadge status={payload.payload.severity} size="sm" /> : null}
        </div>
        {payload.dedup_key ? <p className="mt-2 font-mono text-xs text-muted-foreground">{payload.dedup_key}</p> : null}
      </div>
      {items.length > 1 ? (
        <ol className="space-y-2 border-l border-border pl-4">
          {items.map((ev) => (
            <li key={ev.id} className="text-sm">
              <span className="font-medium">{ev.summary}</span>
              <span className="ml-2 text-xs text-muted-foreground">{formatRelativeTime(ev.receivedAt)}</span>
            </li>
          ))}
        </ol>
      ) : null}
    </div>
  );
}
