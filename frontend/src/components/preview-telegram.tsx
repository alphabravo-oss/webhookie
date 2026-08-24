export type TelegramAction = { actionId: string; value: string; text: string };

type Button = { text?: string; callback_data?: string; url?: string };

export function TelegramPreview({
  body,
  onAction,
  taken,
}: {
  body: string;
  onAction?: (a: TelegramAction) => void;
  taken?: { actionId?: string; text?: string; pending?: boolean };
}) {
  let payload: {
    text?: string;
    chat_id?: string | number;
    reply_markup?: { inline_keyboard?: Button[][] };
  } = {};
  try {
    payload = JSON.parse(body);
  } catch {
    return <p className="text-sm text-muted-foreground">Not JSON</p>;
  }
  const rows = payload.reply_markup?.inline_keyboard ?? [];
  const done = !!taken && !taken.pending;
  return (
    <div className="max-w-sm space-y-2 rounded-2xl bg-[#182533] p-3 text-sm text-zinc-100">
      <p className="whitespace-pre-wrap">{payload.text}</p>
      {done ? (
        <p className="text-[11px] text-[#6ab3f3]">You pressed {taken.text || taken.actionId}</p>
      ) : (
        rows.map((row, i) => (
          <div key={i} className="flex flex-wrap gap-1">
            {row.map((btn, j) => {
              const selected = !!taken && (taken.actionId === btn.callback_data || taken.text === btn.text);
              return (
                <button
                  key={j}
                  type="button"
                  disabled={!!taken || !onAction || !btn.callback_data}
                  aria-pressed={selected}
                  onClick={() =>
                    onAction?.({
                      actionId: btn.callback_data || 'button',
                      value: btn.callback_data || '',
                      text: btn.text || 'Action',
                    })
                  }
                  className={`flex-1 rounded-md bg-[#2b5278] px-3 py-1.5 text-xs text-[#6ab3f3] ${taken?.pending && selected ? 'animate-pulse' : ''}`}
                >
                  {taken?.pending && selected ? '…' : btn.text || btn.callback_data}
                </button>
              );
            })}
          </div>
        ))
      )}
    </div>
  );
}
