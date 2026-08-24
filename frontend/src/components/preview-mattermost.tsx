export type MattermostAction = { actionId: string; value: string; text: string };

type Attachment = {
  title?: string;
  text?: string;
  pretext?: string;
  color?: string;
  actions?: Array<{ id?: string; name?: string; type?: string }>;
};

export function MattermostPreview({
  body,
  onAction,
  taken,
}: {
  body: string;
  onAction?: (a: MattermostAction) => void;
  taken?: { actionId?: string; text?: string; pending?: boolean };
}) {
  let payload: { text?: string; username?: string; attachments?: Attachment[] } = {};
  try {
    payload = JSON.parse(body);
  } catch {
    return <p className="text-sm text-muted-foreground">Not JSON</p>;
  }
  return (
    <div className="space-y-2 rounded-md border border-[#e0e0e0] bg-white p-3 text-sm text-zinc-900">
      {payload.username ? <p className="text-xs font-semibold text-[#1c58d9]">{payload.username} · BOT</p> : null}
      {payload.text ? <p className="whitespace-pre-wrap">{payload.text}</p> : null}
      {(payload.attachments ?? []).map((a, i) => (
        <div key={i} className="flex gap-2">
          <div className="w-1 rounded" style={{ background: a.color ? `#${String(a.color).replace('#', '')}` : '#1c58d9' }} />
          <div className="space-y-1">
            {a.pretext ? <p className="text-xs text-zinc-500">{a.pretext}</p> : null}
            {a.title ? <p className="font-semibold">{a.title}</p> : null}
            {a.text ? <p>{a.text}</p> : null}
            {taken && !taken.pending ? (
              <p className="text-xs text-[#1c58d9]">{taken.text || taken.actionId} by webhookie</p>
            ) : a.actions?.length ? (
              <div className="flex flex-wrap gap-2">
                {a.actions.map((act, j) => {
                  const label = act.name || act.id || 'Action';
                  const selected = !!taken && (taken.actionId === (act.id || label) || taken.text === label);
                  return (
                    <button
                      key={j}
                      type="button"
                      disabled={!!taken || !onAction}
                      aria-pressed={selected}
                      onClick={() => onAction?.({ actionId: act.id || label, value: act.id || label, text: label })}
                      className="rounded bg-[#1c58d9] px-3 py-1 text-xs text-white disabled:opacity-60"
                    >
                      {taken?.pending && selected ? `${label}…` : label}
                    </button>
                  );
                })}
              </div>
            ) : null}
          </div>
        </div>
      ))}
    </div>
  );
}
