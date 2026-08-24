import { useEffect, useState } from 'react';
import { DrawerShell } from '@/components/ui/drawer-shell';
import { ActionButton } from '@/components/ui/action-button';
import { JsonView } from '@/components/json-view';
import { SlackPreview } from '@/components/preview-slack';
import { DiscordPreview } from '@/components/preview-discord';
import { TeamsPreview } from '@/components/preview-teams';
import { PagerDutyPreview } from '@/components/preview-pagerduty';
import { TelegramPreview } from '@/components/preview-telegram';
import { GoogleChatPreview } from '@/components/preview-googlechat';
import { MattermostPreview } from '@/components/preview-mattermost';
import { StatusBadge } from '@/components/ui/status-badge';
import { getEvent, listEvents, replayEvent } from '@/lib/api';
import { eventToCurl } from '@/lib/copy';
import { copyToClipboard } from '@/lib/utils';
import { toastError, toastSuccess } from '@/lib/toast';
import type { Event } from '@/lib/types';

export function Inspector({ id, onClose }: { id: string; onClose: () => void }) {
  const [ev, setEv] = useState<Event | null>(null);
  const [related, setRelated] = useState<Event[]>([]);
  const [tab, setTab] = useState<'body' | 'headers' | 'preview' | 'response'>('preview');
  const [target, setTarget] = useState('');

  useEffect(() => {
    let cancelled = false;
    getEvent(id)
      .then((res) => {
        if (cancelled) return;
        setEv(res.data);
        setTarget(`${window.location.origin}/hooks/generic/default`);
        if (res.data.groupKey) {
          listEvents({ groupKey: res.data.groupKey, limit: 50 }).then((r) => {
            if (!cancelled) setRelated(r.data);
          });
        }
      })
      .catch((err: Error) => toastError(err.message));
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (!ev) return null;

  const origin = window.location.origin;
  const tabs = [
    { id: 'preview' as const, label: 'Preview', enabled: true },
    { id: 'body' as const, label: 'Body', enabled: true },
    { id: 'headers' as const, label: 'Headers', enabled: true },
    { id: 'response' as const, label: 'Response', enabled: true },
  ];

  return (
    <DrawerShell
      title={ev.summary || ev.path}
      subtitle={`${ev.method} ${ev.path}`}
      onClose={onClose}
      actions={<StatusBadge status={ev.valid ? 'success' : 'failed'} label={ev.valid ? 'Valid' : 'Invalid'} size="sm" />}
    >
      <div className="mb-4 flex gap-1 border-b border-border">
        {tabs.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => setTab(t.id)}
            className={`h-9 px-3 text-sm ${tab === t.id ? 'border-b-2 border-primary font-medium' : 'text-muted-foreground'}`}
          >
            {t.label}
          </button>
        ))}
      </div>
      {tab === 'body' && <JsonView value={ev.body} />}
      {tab === 'headers' && <JsonView value={JSON.stringify(ev.headers)} />}
      {tab === 'response' && (
        <div className="space-y-2 text-sm">
          <p>
            Status <span className="font-mono">{ev.status}</span> in {ev.latencyMs}ms
          </p>
          {ev.validationErrors?.length ? (
            <ul className="list-disc pl-5 text-status-error">
              {ev.validationErrors.map((e) => (
                <li key={e.path}>
                  <code>{e.path}</code> {e.message}
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      )}
      {tab === 'preview' && (
        <>
          {ev.provider === 'slack' && <SlackPreview body={ev.body} />}
          {ev.provider === 'discord' && <DiscordPreview body={ev.body} />}
          {ev.provider === 'teams' && <TeamsPreview body={ev.body} />}
          {ev.provider === 'pagerduty' && <PagerDutyPreview body={ev.body} related={related} />}
          {ev.provider === 'telegram' && <TelegramPreview body={ev.body} />}
          {ev.provider === 'googlechat' && <GoogleChatPreview body={ev.body} />}
          {ev.provider === 'mattermost' && <MattermostPreview body={ev.body} />}
          {ev.provider === 'opsgenie' && <p className="text-sm">{ev.summary}</p>}
          {ev.provider === 'generic' && <JsonView value={ev.body} />}
        </>
      )}
      <div className="mt-6 space-y-2">
        <ActionButton
          size="sm"
          onClick={() => {
            void copyToClipboard(eventToCurl(origin, ev)).then((ok) =>
              ok ? toastSuccess('Copied cURL') : toastError('Copy failed'),
            );
          }}
        >
          Copy as cURL
        </ActionButton>
        <div className="flex gap-2">
          <input
            className="h-9 flex-1 rounded-md border border-border bg-background px-3 text-sm"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder="Replay target URL"
          />
          <ActionButton
            size="sm"
            intent="primary"
            onClick={() => {
              replayEvent(ev.id, target)
                .then(() => toastSuccess('Replayed'))
                .catch((err: Error) => toastError(err.message));
            }}
          >
            Replay
          </ActionButton>
        </div>
      </div>
    </DrawerShell>
  );
}
