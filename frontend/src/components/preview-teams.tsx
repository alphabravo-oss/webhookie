export type CardAction = { actionId: string; value: string; text: string };

function collectActions(node: unknown, acc: CardAction[]) {
  if (!node || typeof node !== 'object') return;
  const o = node as Record<string, unknown>;
  if (typeof o.type === 'string' && String(o.type).startsWith('Action.') && o.title) {
    acc.push({
      actionId: String(o.id ?? o.type),
      value: JSON.stringify(o.data ?? {}),
      text: String(o.title),
    });
  }
  for (const v of Object.values(o)) {
    if (Array.isArray(v)) v.forEach((x) => collectActions(x, acc));
    else collectActions(v, acc);
  }
}

export function TeamsPreview({
  body,
  onAction,
  taken,
}: {
  body: string;
  onAction?: (a: CardAction) => void;
  taken?: { actionId?: string; text?: string; pending?: boolean };
}) {
  let payload: Record<string, unknown> = {};
  try {
    payload = JSON.parse(body);
  } catch {
    return <p className="text-sm text-muted-foreground">Not JSON</p>;
  }
  if (payload['@type'] === 'MessageCard') {
    const color = String(payload.themeColor ?? '0078d4');
    const sections = (payload.sections as Array<{ facts?: Array<{ name: string; value: string }>; text?: string }> | undefined) ?? [];
    const potential = (payload.potentialAction as Array<{ name?: string; '@type'?: string }> | undefined) ?? [];
    return (
      <div className="flex overflow-hidden rounded-lg border border-border bg-white text-zinc-900 shadow-sm">
        <div className="w-1" style={{ background: `#${color.replace('#', '')}` }} />
        <div className="space-y-2 p-4">
          {payload.title ? <h3 className="font-semibold">{String(payload.title)}</h3> : null}
          {payload.text ? <p className="text-sm text-zinc-600">{String(payload.text)}</p> : null}
          {sections.map((s, i) => (
            <div key={i}>
              {s.text ? <p className="text-sm">{s.text}</p> : null}
              {s.facts ? (
                <dl className="grid grid-cols-2 gap-1 text-xs">
                  {s.facts.map((f) => (
                    <div key={f.name}>
                      <dt className="text-zinc-500">{f.name}</dt>
                      <dd>{f.value}</dd>
                    </div>
                  ))}
                </dl>
              ) : null}
            </div>
          ))}
          {taken && !taken.pending ? (
            <p className="mt-2 text-xs text-[#5B5FC7]">Your response was sent to the app · {taken.text || taken.actionId}</p>
          ) : potential.length ? (
            <div className="mt-2 flex flex-wrap gap-2">
              {potential.map((a, i) => {
                const label = a.name || 'Action';
                const selected = !!taken && (taken.actionId === label || taken.text === label);
                return (
                  <button
                    key={label + i}
                    type="button"
                    disabled={!!taken || !onAction}
                    aria-pressed={selected}
                    onClick={() => onAction?.({ actionId: label, value: label, text: label })}
                    className="rounded bg-[#5B5FC7] px-3 py-1 text-xs text-white disabled:opacity-60"
                  >
                    {taken?.pending && selected ? 'Sending…' : label}
                  </button>
                );
              })}
            </div>
          ) : null}
        </div>
      </div>
    );
  }
  const atts = (payload.attachments as Array<{ content?: { body?: Array<{ type?: string; text?: string }> } }> | undefined) ?? [];
  const texts = atts.flatMap((a) => (a.content?.body ?? []).map((b) => b.text).filter(Boolean));
  const actions: CardAction[] = [];
  collectActions(payload, actions);
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <p className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">Adaptive Card</p>
      {texts.length ? texts.map((t, i) => <p key={i}>{t}</p>) : <p className="text-sm text-muted-foreground">Adaptive Card payload</p>}
      {taken && !taken.pending ? (
        <p className="mt-3 text-xs text-[#5059c9]">Your response was sent to the app · {taken.text || taken.actionId}</p>
      ) : actions.length ? (
        <div className="mt-3 flex flex-wrap gap-2">
          {actions.map((a) => {
            const selected = !!taken && (taken.actionId === a.actionId || taken.text === a.text);
            return (
              <button
                key={a.text + a.actionId}
                type="button"
                disabled={!!taken || !onAction}
                aria-pressed={selected}
                onClick={() => onAction?.(a)}
                className="rounded bg-[#5059c9] px-3 py-1 text-xs text-white disabled:opacity-60"
              >
                {taken?.pending && selected ? 'Sending…' : a.text}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
