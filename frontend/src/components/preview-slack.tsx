export type SlackAction = { actionId: string; value: string; text: string; blockId: string };

type Block = {
  type?: string;
  block_id?: string;
  text?: { text?: string } | string;
  image_url?: string;
  elements?: Array<{ text?: { text?: string } | string; type?: string; action_id?: string; value?: string; style?: string }>;
};

function textOf(t: unknown): string {
  if (!t) return '';
  if (typeof t === 'string') return t;
  if (typeof t === 'object' && t && 'text' in t) return String((t as { text?: string }).text ?? '');
  return '';
}

function mrkdwn(s: string) {
  return s.replace(/\*(.*?)\*/g, '$1').replace(/_(.*?)_/g, '$1').replace(/`(.*?)`/g, '$1');
}

export function SlackPreview({
  body,
  onAction,
  taken,
}: {
  body: string;
  onAction?: (a: SlackAction) => void;
  taken?: { actionId?: string; text?: string; pending?: boolean };
}) {
  let payload: { text?: string; blocks?: Block[]; attachments?: Array<{ color?: string; pretext?: string; text?: string }> } = {};
  try {
    payload = JSON.parse(body);
  } catch {
    return <p className="text-sm text-muted-foreground">Not JSON</p>;
  }
  const blocks = payload.blocks ?? [];
  return (
    <div className="space-y-2 rounded-lg border border-zinc-800 bg-[#1a1d21] p-4 text-sm text-zinc-100">
      {payload.text && blocks.length === 0 ? <p>{mrkdwn(payload.text)}</p> : null}
      {blocks.map((b, i) => {
        if (b.type === 'divider') return <hr key={i} className="border-zinc-700" />;
        if (b.type === 'header') return <h3 key={i} className="text-base font-semibold">{mrkdwn(textOf(b.text))}</h3>;
        if (b.type === 'section') return <p key={i}>{mrkdwn(textOf(b.text))}</p>;
        if (b.type === 'context') {
          return (
            <p key={i} className="text-xs text-zinc-400">
              {(b.elements ?? []).map((el) => mrkdwn(textOf(el.text ?? el))).join(' · ')}
            </p>
          );
        }
        if (b.type === 'image' && b.image_url) return <img key={i} src={b.image_url} alt="" className="max-h-40 rounded" />;
        if (b.type === 'actions') {
          return (
            <div key={i} className="space-y-2">
              <div className="flex flex-wrap gap-2">
                {(b.elements ?? []).map((el, j) => {
                  const label = textOf(el.text ?? el) || el.type || 'Action';
                  const selected =
                    !!taken &&
                    (taken.actionId === (el.action_id || 'button') || taken.text === label);
                  const locked = !!taken || !onAction;
                  return (
                    <button
                      key={j}
                      type="button"
                      disabled={locked}
                      aria-pressed={selected}
                      onClick={() =>
                        onAction?.({
                          actionId: el.action_id || 'button',
                          value: el.value || label,
                          text: label,
                          blockId: b.block_id || '',
                        })
                      }
                      className={`rounded px-3 py-1 text-xs ${
                        selected
                          ? 'bg-[#2bac76] text-white'
                          : el.style === 'danger'
                            ? 'bg-red-700'
                            : 'bg-[#1264a3]'
                      } ${locked && !selected ? 'bg-zinc-700 opacity-50' : ''} disabled:cursor-default`}
                    >
                      {selected && !taken?.pending ? `✓ ${label}` : taken?.pending && selected ? `${label}…` : label}
                    </button>
                  );
                })}
              </div>
              {taken && !taken.pending ? (
                <p className="text-xs text-zinc-400">webhookie clicked {taken.text || taken.actionId}</p>
              ) : null}
            </div>
          );
        }
        return null;
      })}
      {(payload.attachments ?? []).map((a, i) => (
        <div key={i} className="flex gap-2">
          <div className="w-1 rounded" style={{ background: a.color ? `#${a.color.replace('#', '')}` : '#439fe0' }} />
          <div>
            {a.pretext ? <p className="text-xs text-zinc-400">{a.pretext}</p> : null}
            {a.text ? <p>{a.text}</p> : null}
          </div>
        </div>
      ))}
    </div>
  );
}
