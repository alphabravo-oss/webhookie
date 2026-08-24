type Embed = {
  title?: string;
  description?: string;
  color?: number;
  fields?: Array<{ name: string; value: string }>;
  footer?: { text?: string };
};

export type DiscordAction = { actionId: string; value: string; text: string };

export function DiscordPreview({
  body,
  onAction,
  taken,
}: {
  body: string;
  onAction?: (a: DiscordAction) => void;
  taken?: { actionId?: string; text?: string; pending?: boolean };
}) {
  let payload: {
    content?: string;
    embeds?: Embed[];
    username?: string;
    components?: Array<{ components?: Array<{ type?: number; label?: string; custom_id?: string; style?: number }> }>;
  } = {};
  try {
    payload = JSON.parse(body);
  } catch {
    return <p className="text-sm text-muted-foreground">Not JSON</p>;
  }
  return (
    <div className="space-y-3 rounded-lg border border-border bg-[#313338] p-4 text-sm text-zinc-100">
      {payload.username ? <p className="text-xs font-medium text-zinc-400">{payload.username}</p> : null}
      {payload.content ? <p>{payload.content}</p> : null}
      {(payload.embeds ?? []).map((e, i) => (
        <div key={i} className="flex overflow-hidden rounded bg-[#2b2d31]">
          <div className="w-1" style={{ background: e.color != null ? `#${e.color.toString(16).padStart(6, '0')}` : '#5865f2' }} />
          <div className="space-y-1 p-3">
            {e.title ? <p className="font-semibold">{e.title}</p> : null}
            {e.description ? <p className="text-zinc-300">{e.description}</p> : null}
            {e.fields?.length ? (
              <dl className="grid grid-cols-2 gap-2 text-xs">
                {e.fields.map((f) => (
                  <div key={f.name}>
                    <dt className="font-medium text-zinc-400">{f.name}</dt>
                    <dd>{f.value}</dd>
                  </div>
                ))}
              </dl>
            ) : null}
            {e.footer?.text ? <p className="text-xs text-zinc-500">{e.footer.text}</p> : null}
          </div>
        </div>
      ))}
      <div className="flex flex-wrap gap-2">
        {(payload.components ?? []).flatMap((row) => row.components ?? []).map((c, i) => {
          const selected = !!taken && (taken.actionId === (c.custom_id || 'button') || taken.text === c.label);
          const style =
            c.style === 4 ? 'bg-[#da373c]' : c.style === 3 ? 'bg-[#248046]' : c.style === 2 ? 'bg-[#4e5058]' : 'bg-[#5865f2]';
          return (
            <button
              key={i}
              type="button"
              disabled={!!taken || !onAction}
              aria-pressed={selected}
              onClick={() =>
                onAction?.({ actionId: c.custom_id || 'button', value: c.custom_id || '', text: c.label || 'Action' })
              }
              className={`rounded px-3 py-1 text-xs text-white ${style} ${taken ? 'opacity-50' : ''} ${taken?.pending && selected ? 'animate-pulse' : ''}`}
            >
              {taken?.pending && selected ? `${c.label || c.custom_id}…` : c.label || c.custom_id}
            </button>
          );
        })}
      </div>
      {taken && !taken.pending ? (
        <p className="text-xs italic text-zinc-400">webhookie used {taken.text || taken.actionId}</p>
      ) : null}
    </div>
  );
}
