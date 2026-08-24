import { useEffect, useMemo, useState } from 'react';
import { useRouter } from '@/lib/navigation';
import { Copy, Hash, Plus, Siren } from 'lucide-react';
import { PageHeader, PageShell } from '@/components/ui/page';
import { ActionButton } from '@/components/ui/action-button';
import { SlackPreview } from '@/components/preview-slack';
import { TeamsPreview } from '@/components/preview-teams';
import { DiscordPreview } from '@/components/preview-discord';
import { TelegramPreview } from '@/components/preview-telegram';
import { GoogleChatPreview } from '@/components/preview-googlechat';
import { MattermostPreview } from '@/components/preview-mattermost';
import { StatusBadge } from '@/components/ui/status-badge';
import {
  createChannel,
  getWorkspace,
  listEvents,
  listInteractions,
  patchWorkspace,
  postAction,
} from '@/lib/api';
import { copyToClipboard, formatRelativeTime } from '@/lib/utils';
import { toastError, toastSuccess } from '@/lib/toast';
import type { Channel, Event, Interaction, TakenAction, Workspace } from '@/lib/types';
import { cn } from '@/lib/utils';

const workspaceId: Record<string, string> = {
  slack: 'ws-slack',
  teams: 'ws-teams',
  discord: 'ws-discord',
  pagerduty: 'ws-pagerduty',
  telegram: 'ws-telegram',
  googlechat: 'ws-googlechat',
  mattermost: 'ws-mattermost',
  opsgenie: 'ws-opsgenie',
};

const chrome: Record<
  string,
  {
    workspace: string;
    rail: string;
    railTitle: string;
    railLabel: string;
    railBtn: string;
    railBtnActive: string;
    pane: string;
    header: string;
    headerText: string;
    input: string;
    composer: string;
  }
> = {
  slack: {
    workspace: 'Webhookie',
    rail: 'bg-[#3F0E40] text-[#d5c4d8]',
    railTitle: 'text-white',
    railLabel: 'text-[#ab9ba9]',
    railBtn: 'hover:bg-white/10',
    railBtnActive: 'bg-[#1164A3] text-white',
    pane: 'bg-[#1a1d21] text-zinc-100',
    header: 'border-white/10',
    headerText: 'text-zinc-100',
    input: 'border-white/15 bg-white/10 text-white placeholder:text-white/40',
    composer: 'border-white/10 bg-[#222529] text-zinc-400',
  },
  teams: {
    workspace: 'Webhookie',
    rail: 'bg-[#5B5FC7] text-white',
    railTitle: 'text-white',
    railLabel: 'text-white/70',
    railBtn: 'hover:bg-white/15',
    railBtnActive: 'bg-white/20 text-white',
    pane: 'bg-[#f5f5f5] text-zinc-900',
    header: 'border-[#e0e0e0] bg-white',
    headerText: 'text-zinc-900',
    input: 'border-white/30 bg-white/15 text-white placeholder:text-white/60',
    composer: 'border-[#e0e0e0] bg-white text-zinc-500',
  },
  discord: {
    workspace: 'Webhookie',
    rail: 'bg-[#2b2d31] text-[#dbdee1]',
    railTitle: 'text-white',
    railLabel: 'text-[#949ba4]',
    railBtn: 'hover:bg-white/5',
    railBtnActive: 'bg-[#404249] text-white',
    pane: 'bg-[#313338] text-[#dbdee1]',
    header: 'border-black/40',
    headerText: 'text-white',
    input: 'border-white/10 bg-[#1e1f22] text-[#dbdee1] placeholder:text-[#949ba4]',
    composer: 'border-black/30 bg-[#383a40] text-[#949ba4]',
  },
  pagerduty: {
    workspace: 'Webhookie',
    rail: 'bg-[#1C2333] text-[#c5cad3]',
    railTitle: 'text-white',
    railLabel: 'text-[#8b93a1]',
    railBtn: 'hover:bg-white/10',
    railBtnActive: 'bg-[#06AC38] text-white',
    pane: 'bg-[#f4f5f7] text-zinc-900',
    header: 'border-[#e2e5ea] bg-white',
    headerText: 'text-zinc-900',
    input: 'border-white/15 bg-white/10 text-white placeholder:text-white/40',
    composer: 'border-[#e2e5ea] bg-white text-zinc-500',
  },
  telegram: {
    workspace: 'Webhookie',
    rail: 'bg-[#17212b] text-[#8b9aab]',
    railTitle: 'text-white',
    railLabel: 'text-[#6d7f8f]',
    railBtn: 'hover:bg-white/5',
    railBtnActive: 'bg-[#2b5278] text-white',
    pane: 'bg-[#0e1621] text-zinc-100',
    header: 'border-white/10',
    headerText: 'text-zinc-100',
    input: 'border-white/10 bg-[#242f3d] text-white placeholder:text-white/40',
    composer: 'border-white/10 bg-[#17212b] text-[#6d7f8f]',
  },
  googlechat: {
    workspace: 'Webhookie',
    rail: 'bg-[#e8f0fe] text-[#1a73e8]',
    railTitle: 'text-[#202124]',
    railLabel: 'text-[#5f6368]',
    railBtn: 'hover:bg-white/80',
    railBtnActive: 'bg-white text-[#1a73e8] shadow-sm',
    pane: 'bg-[#f8f9fa] text-zinc-900',
    header: 'border-[#dadce0] bg-white',
    headerText: 'text-zinc-900',
    input: 'border-[#dadce0] bg-white text-zinc-900 placeholder:text-zinc-400',
    composer: 'border-[#dadce0] bg-white text-zinc-500',
  },
  mattermost: {
    workspace: 'Webhookie',
    rail: 'bg-[#1e325c] text-[#a4b4cc]',
    railTitle: 'text-white',
    railLabel: 'text-[#7b8aa5]',
    railBtn: 'hover:bg-white/10',
    railBtnActive: 'bg-[#166de0] text-white',
    pane: 'bg-[#ffffff] text-zinc-900',
    header: 'border-[#e0e0e0] bg-white',
    headerText: 'text-zinc-900',
    input: 'border-white/15 bg-white/10 text-white placeholder:text-white/40',
    composer: 'border-[#e0e0e0] bg-[#f4f4f4] text-zinc-500',
  },
  opsgenie: {
    workspace: 'Webhookie',
    rail: 'bg-[#172B4D] text-[#c1c7d0]',
    railTitle: 'text-white',
    railLabel: 'text-[#8b93a1]',
    railBtn: 'hover:bg-white/10',
    railBtnActive: 'bg-[#36B37E] text-white',
    pane: 'bg-[#f4f5f7] text-zinc-900',
    header: 'border-[#e2e5ea] bg-white',
    headerText: 'text-zinc-900',
    input: 'border-white/15 bg-white/10 text-white placeholder:text-white/40',
    composer: 'border-[#e2e5ea] bg-white text-zinc-500',
  },
};

export function MockWorkspace({
  provider,
  channelId,
}: {
  provider: string;
  channelId?: string;
}) {
  const router = useRouter();
  const [ws, setWs] = useState<Workspace | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [interactions, setInteractions] = useState<Interaction[]>([]);
  const [name, setName] = useState('');
  const [callback, setCallback] = useState('');
  const [secret, setSecret] = useState('');
  const [pending, setPending] = useState<TakenAction & { eventId: string } | null>(null);

  const id = workspaceId[provider];
  const theme = chrome[provider] ?? chrome.slack;
  const channels = ws?.channels ?? [];
  const active = useMemo(
    () => channels.find((c) => c.id === channelId) ?? channels[0],
    [channels, channelId],
  );

  const reload = () => {
    getWorkspace(id)
      .then((res) => {
        setWs(res.data);
        setCallback(res.data.interactivityUrl);
        setSecret(res.data.signingSecret);
        if (!channelId && res.data.channels[0]) {
          router.push(`/${provider}/${res.data.channels[0].id}`);
        }
      })
      .catch((err: Error) => toastError(err.message));
  };

  useEffect(() => {
    reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  useEffect(() => {
    if (!active || !ws) return;
    const load = () =>
      Promise.all([
        listEvents({ sinkId: active.sinkId, limit: 100 }),
        listInteractions(ws.id, active.id),
      ])
        .then(([ev, ix]) => {
          setEvents([...(ev.data ?? [])].reverse());
          setInteractions(ix.data ?? []);
        })
        .catch((err: Error) => toastError(err.message));
    load();
    const es = new EventSource('/api/v1/events/stream');
    es.addEventListener('webhook', (msg) => {
      const ev = JSON.parse((msg as MessageEvent).data) as Event;
      if (ev.sinkId === active.sinkId) {
        setEvents((prev) => [...prev.filter((e) => e.id !== ev.id), ev]);
      }
    });
    return () => es.close();
  }, [active?.sinkId, active, ws?.id]);

  const noun =
    provider === 'pagerduty' ? 'service' : provider === 'telegram' ? 'chat' : provider === 'googlechat' ? 'space' : provider === 'opsgenie' ? 'team' : 'channel';
  const prefix = provider === 'slack' || provider === 'discord' || provider === 'mattermost' ? '#' : '';
  const darkCopy = provider === 'slack' || provider === 'discord' || provider === 'telegram';
  const incident = provider === 'pagerduty' || provider === 'opsgenie';

  const takenByEvent = useMemo(() => {
    const map = new Map<string, TakenAction>();
    for (const ix of [...interactions].reverse()) {
      if (!ix.eventId) continue;
      map.set(ix.eventId, takenFromInteraction(ix));
    }
    return map;
  }, [interactions]);

  const takenFor = (eventId: string): TakenAction | undefined => {
    if (pending?.eventId === eventId) return pending;
    return takenByEvent.get(eventId);
  };

  const act = (eventId: string, kind: string, extra: { actionId?: string; value?: string; text?: string } = {}) => {
    if (!ws || !active) return;
    setPending({ eventId, kind, actionId: extra.actionId, text: extra.text, pending: true });
    postAction(ws.id, active.id, { eventId, kind, ...extra })
      .then((res) => {
        if (res.data.target) toastSuccess(`Posted ${kind} to interactivity URL`);
        else toastSuccess(`${kind} recorded — set an interactivity URL to deliver it`);
        return Promise.all([
          listEvents({ sinkId: active.sinkId, limit: 100 }),
          listInteractions(ws.id, active.id),
        ]);
      })
      .then((res) => {
        if (!res) return;
        const [ev, ix] = res;
        setEvents([...(ev.data ?? [])].reverse());
        setInteractions(ix.data ?? []);
        setPending(null);
      })
      .catch((err: Error) => {
        setPending(null);
        toastError(err.message);
      });
  };

  return (
    <PageShell className="space-y-4">
      <PageHeader
        eyebrow="Mock"
        title={ws?.name ?? provider}
        description={`Create ${noun}s, copy their webhook URLs, and interact as an operator would.`}
      />
      <div className="grid min-h-[32rem] grid-cols-1 overflow-hidden rounded-lg border border-border lg:grid-cols-[220px_1fr]">
        <aside className={cn('border-b p-3 lg:border-b-0 lg:border-r lg:border-black/10', theme.rail)}>
          <p className={cn('mb-3 px-2 text-xs font-semibold', theme.railTitle)}>{theme.workspace}</p>
          <p className={cn('mb-2 px-2 text-[10px] font-semibold uppercase tracking-wider', theme.railLabel)}>
            {noun}s
          </p>
          <div className="space-y-0.5">
            {channels.map((c) => (
              <ChannelRow
                key={c.id}
                provider={provider}
                channel={c}
                active={active?.id === c.id}
                prefix={prefix}
                theme={theme}
              />
            ))}
          </div>
          <form
            className="mt-3 flex gap-1"
            onSubmit={(e) => {
              e.preventDefault();
              if (!name.trim()) return;
              createChannel(id, name.trim())
                .then((res) => {
                  toastSuccess(`Created ${res.data.name}`);
                  setName('');
                  reload();
                  router.push(`/${provider}/${res.data.id}`);
                })
                .catch((err: Error) => toastError(err.message));
            }}
          >
            <input
              className={cn('h-8 min-w-0 flex-1 rounded-md border px-2 text-xs', theme.input)}
              placeholder={`New ${noun}`}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <ActionButton size="icon" type="submit" icon={<Plus className="h-3.5 w-3.5" />} />
          </form>
        </aside>
        <section className={cn('flex min-h-0 flex-col', theme.pane)}>
          {active ? (
            <>
              <header className={cn('flex flex-wrap items-center gap-2 border-b px-4 py-3', theme.header)}>
                <h2 className={cn('text-sm font-semibold', theme.headerText)}>
                  {prefix}
                  {active.name}
                </h2>
                <code className="max-w-full truncate font-mono text-[11px] opacity-70">{active.url || active.path}</code>
                <ActionButton
                  size="sm"
                  intent="ghost"
                  className={darkCopy ? 'text-zinc-300 hover:bg-white/10 hover:text-white' : ''}
                  icon={<Copy className="h-3.5 w-3.5" />}
                  onClick={() => {
                    const url = active.url || `${window.location.origin}${active.path}`;
                    void copyToClipboard(url).then((ok) => (ok ? toastSuccess('Copied URL') : toastError('Copy failed')));
                  }}
                >
                  Copy URL
                </ActionButton>
              </header>
              <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
                {events.length === 0 ? (
                  <p className="text-sm opacity-70">
                    No messages yet. POST to this {noun}’s webhook URL.
                  </p>
                ) : incident ? (
                  <IncidentList
                    events={events}
                    resolveLabel={provider === 'opsgenie' ? 'Close' : 'Resolve'}
                    pending={pending}
                    onAck={(id, key) => act(id, 'ack', { value: key, text: 'Acknowledge' })}
                    onResolve={(id, key) =>
                      act(id, provider === 'opsgenie' ? 'close' : 'resolve', {
                        value: key,
                        text: provider === 'opsgenie' ? 'Close' : 'Resolve',
                      })
                    }
                  />
                ) : (
                  events.map((ev) => (
                    <article key={ev.id} className="flex gap-3">
                      <BotMark provider={provider} />
                      <div className="min-w-0 flex-1 space-y-1">
                        <p className="text-[11px] opacity-60">
                          webhookie · {formatRelativeTime(ev.receivedAt)}
                          {ev.valid ? '' : ' · invalid'}
                        </p>
                        {provider === 'slack' && (
                          <SlackPreview body={ev.body} taken={takenFor(ev.id)} onAction={(a) => act(ev.id, 'button', a)} />
                        )}
                        {provider === 'teams' && (
                          <TeamsPreview body={ev.body} taken={takenFor(ev.id)} onAction={(a) => act(ev.id, 'submit', a)} />
                        )}
                        {provider === 'discord' && (
                          <DiscordPreview body={ev.body} taken={takenFor(ev.id)} onAction={(a) => act(ev.id, 'button', a)} />
                        )}
                        {provider === 'telegram' && (
                          <TelegramPreview body={ev.body} taken={takenFor(ev.id)} onAction={(a) => act(ev.id, 'button', a)} />
                        )}
                        {provider === 'googlechat' && (
                          <GoogleChatPreview body={ev.body} taken={takenFor(ev.id)} onAction={(a) => act(ev.id, 'submit', a)} />
                        )}
                        {provider === 'mattermost' && (
                          <MattermostPreview
                            body={ev.body}
                            taken={takenFor(ev.id)}
                            onAction={(a) => act(ev.id, 'button', a)}
                          />
                        )}
                      </div>
                    </article>
                  ))
                )}
              </div>
              <div className={cn('border-t px-4 py-3 text-xs', theme.composer)}>
                Message as this {noun}’s incoming webhook — clicks fire two-way callbacks below.
              </div>
            </>
          ) : (
            <p className="p-6 text-sm opacity-70">Create a {noun} to get a webhook URL.</p>
          )}
        </section>
      </div>
      <div className="rounded-lg border border-border bg-card p-4">
        <p className="mb-2 text-sm font-semibold">Two-way callbacks</p>
        <p className="mb-3 text-xs text-muted-foreground">
          When you click Approve / Ack, webhookie POSTs a provider-shaped interaction payload here.
        </p>
        <div className="grid gap-2 sm:grid-cols-2">
          <label className="text-xs">
            Interactivity URL
            <input
              className="mt-1 block h-9 w-full rounded-md border border-border bg-background px-3 font-mono text-xs"
              value={callback}
              onChange={(e) => setCallback(e.target.value)}
              placeholder="https://your-app/interactions"
            />
          </label>
          <label className="text-xs">
            Signing secret
            <input
              className="mt-1 block h-9 w-full rounded-md border border-border bg-background px-3 font-mono text-xs"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              placeholder="optional"
            />
          </label>
        </div>
        <ActionButton
          className="mt-3"
          size="sm"
          onClick={() => {
            patchWorkspace(id, { interactivityUrl: callback, signingSecret: secret })
              .then(() => toastSuccess('Saved'))
              .catch((err: Error) => toastError(err.message));
          }}
        >
          Save
        </ActionButton>
        {interactions.length > 0 ? (
          <ol className="mt-4 space-y-1 border-t border-border pt-3 font-mono text-[11px] text-muted-foreground">
            {interactions.slice(0, 8).map((ix) => (
              <li key={ix.id}>
                {ix.kind}
                {ix.actionId ? ` · ${ix.actionId}` : ''}
                {ix.target ? ` → ${ix.status ?? 'queued'}` : ' · recorded locally'}
              </li>
            ))}
          </ol>
        ) : null}
      </div>
    </PageShell>
  );
}

function takenFromInteraction(ix: Interaction): TakenAction {
  let text = ix.actionId;
  try {
    const params = new URLSearchParams(ix.payload);
    const raw = params.get('payload') || ix.payload;
    const p = JSON.parse(raw) as {
      actions?: Array<{ text?: { text?: string } }>;
      callback_query?: { data?: string };
      context?: { action?: string };
      action?: { actionMethodName?: string };
    };
    text =
      p.actions?.[0]?.text?.text ||
      p.callback_query?.data ||
      p.context?.action ||
      p.action?.actionMethodName ||
      ix.actionId;
  } catch {
    /* payload is not JSON */
  }
  return { actionId: ix.actionId, kind: ix.kind, text };
}

function BotMark({ provider }: { provider: string }) {
  const bg =
    provider === 'slack'
      ? 'bg-[#1264a3]'
      : provider === 'teams'
        ? 'bg-[#5B5FC7]'
        : provider === 'telegram'
          ? 'bg-[#229ED9]'
          : provider === 'googlechat'
            ? 'bg-[#1a73e8]'
            : provider === 'mattermost'
              ? 'bg-[#1c58d9]'
              : 'bg-[#5865f2]';
  return (
    <span className={cn('mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-[10px] font-bold text-white', bg)}>
      W
    </span>
  );
}

function ChannelRow({
  provider,
  channel,
  active,
  prefix,
  theme,
}: {
  provider: string;
  channel: Channel;
  active: boolean;
  prefix: string;
  theme: (typeof chrome)[string];
}) {
  const router = useRouter();
  const Icon = provider === 'pagerduty' || provider === 'opsgenie' ? Siren : Hash;
  return (
    <button
      type="button"
      onClick={() => router.push(`/${provider}/${channel.id}`)}
      className={cn(
        'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm',
        active ? theme.railBtnActive : theme.railBtn,
      )}
    >
      <Icon className="h-3.5 w-3.5" />
      {prefix}
      {channel.name}
    </button>
  );
}

function IncidentList({
  events,
  onAck,
  onResolve,
  resolveLabel = 'Resolve',
  pending,
}: {
  events: Event[];
  onAck: (id: string, key: string) => void;
  onResolve: (id: string, key: string) => void;
  resolveLabel?: string;
  pending?: (TakenAction & { eventId: string }) | null;
}) {
  const groups = new Map<string, Event[]>();
  for (const ev of events) {
    const key = ev.groupKey || ev.id;
    groups.set(key, [...(groups.get(key) ?? []), ev]);
  }
  return (
    <div className="space-y-3">
      {[...groups.entries()].map(([key, items]) => {
        const latest = items[items.length - 1];
        const opened = items[0];
        const resolved =
          latest.summary.toLowerCase().includes('resolve') || latest.summary.toLowerCase().includes('close');
        const acked = latest.summary.toLowerCase().includes('ack');
        let severity = 'error';
        try {
          const body = JSON.parse(opened.body) as { payload?: { severity?: string }; priority?: string };
          severity = body.payload?.severity ?? body.priority ?? 'error';
        } catch {
          /* ignore */
        }
        return (
          <div key={key} className="overflow-hidden rounded-lg border border-[#e2e5ea] bg-white shadow-sm">
            <div className={cn('h-1', resolved ? 'bg-[#06AC38]' : acked ? 'bg-[#f5c518]' : 'bg-[#e74c3c]')} />
            <div className="p-4">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <p className="font-medium">{opened.summary.replace(/^(trigger|acknowledge|resolve|create|close) · /i, '')}</p>
                  <p className="font-mono text-[11px] text-zinc-500">{key}</p>
                </div>
                <div className="flex gap-1">
                  <StatusBadge status={severity} size="sm" />
                  <StatusBadge
                    status={resolved ? (resolveLabel === 'Close' ? 'closed' : 'resolved') : acked ? 'acknowledged' : 'triggered'}
                    size="sm"
                  />
                </div>
              </div>
              <ol className="mt-3 space-y-1 border-l border-zinc-200 pl-3 text-xs text-zinc-500">
                {items.map((ev) => (
                  <li key={ev.id}>
                    {ev.summary} · {formatRelativeTime(ev.receivedAt)}
                  </li>
                ))}
              </ol>
              {resolved ? null : (
                <div className="mt-3 flex gap-2">
                  {acked ? null : (
                    <ActionButton
                      size="sm"
                      disabled={pending?.kind === 'ack' && pending.eventId === latest.id}
                      onClick={() => onAck(latest.id, key)}
                    >
                      {pending?.kind === 'ack' && pending.eventId === latest.id ? 'Acknowledging…' : 'Acknowledge'}
                    </ActionButton>
                  )}
                  <ActionButton
                    size="sm"
                    intent="primary"
                    disabled={pending?.eventId === latest.id && (pending.kind === 'resolve' || pending.kind === 'close')}
                    onClick={() => onResolve(latest.id, key)}
                  >
                    {pending?.eventId === latest.id && (pending.kind === 'resolve' || pending.kind === 'close')
                      ? `${resolveLabel}…`
                      : resolveLabel}
                  </ActionButton>
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
