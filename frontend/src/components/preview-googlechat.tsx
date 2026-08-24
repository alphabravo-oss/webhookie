export type GoogleChatAction = { actionId: string; value: string; text: string };

function collectButtons(node: unknown, acc: GoogleChatAction[]) {
  if (!node || typeof node !== 'object') return;
  const o = node as Record<string, unknown>;
  if (typeof o.text === 'string' && o.onClick && typeof o.onClick === 'object') {
    const click = o.onClick as { action?: { function?: string; actionMethodName?: string }; openLink?: { url?: string } };
    const fn = click.action?.function || click.action?.actionMethodName || click.openLink?.url || 'action';
    acc.push({ actionId: fn, value: fn, text: String(o.text) });
  }
  for (const v of Object.values(o)) {
    if (Array.isArray(v)) v.forEach((x) => collectButtons(x, acc));
    else collectButtons(v, acc);
  }
}

function collectText(node: unknown, acc: string[]) {
  if (!node || typeof node !== 'object') return;
  const o = node as Record<string, unknown>;
  if (o.textParagraph && typeof o.textParagraph === 'object') {
    const t = (o.textParagraph as { text?: string }).text;
    if (t) acc.push(t.replace(/<[^>]+>/g, ''));
  }
  if (o.decoratedText && typeof o.decoratedText === 'object') {
    const t = (o.decoratedText as { text?: string }).text;
    if (t) acc.push(t.replace(/<[^>]+>/g, ''));
  }
  for (const v of Object.values(o)) {
    if (Array.isArray(v)) v.forEach((x) => collectText(x, acc));
    else collectText(v, acc);
  }
}

export function GoogleChatPreview({
  body,
  onAction,
  taken,
}: {
  body: string;
  onAction?: (a: GoogleChatAction) => void;
  taken?: { actionId?: string; text?: string; pending?: boolean };
}) {
  let payload: {
    text?: string;
    cardsV2?: Array<{ card?: { header?: { title?: string; subtitle?: string } } }>;
    cards?: unknown[];
  } = {};
  try {
    payload = JSON.parse(body);
  } catch {
    return <p className="text-sm text-muted-foreground">Not JSON</p>;
  }
  const actions: GoogleChatAction[] = [];
  collectButtons(payload, actions);
  const texts: string[] = [];
  collectText(payload, texts);
  const title = payload.cardsV2?.[0]?.card?.header?.title;
  return (
    <div className="space-y-2 rounded-lg border border-[#dadce0] bg-white p-4 text-sm text-zinc-900 shadow-sm">
      {title ? <p className="font-semibold">{title}</p> : null}
      {payload.text ? <p>{payload.text}</p> : null}
      {texts.map((t, i) => (
        <p key={i} className="text-zinc-700">
          {t}
        </p>
      ))}
      {taken && !taken.pending ? (
        <p className="mt-2 rounded bg-[#e6f4ea] px-2 py-1 text-xs text-[#137333]">{taken.text || taken.actionId}</p>
      ) : actions.length ? (
        <div className="mt-2 flex flex-wrap gap-2">
          {actions.map((a) => {
            const selected = !!taken && (taken.actionId === a.actionId || taken.text === a.text);
            return (
              <button
                key={a.text + a.actionId}
                type="button"
                disabled={!!taken || !onAction}
                aria-pressed={selected}
                onClick={() => onAction?.(a)}
                className="rounded border border-[#1a73e8] px-3 py-1 text-xs font-medium text-[#1a73e8] disabled:opacity-60"
              >
                {taken?.pending && selected ? 'Working…' : a.text}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
