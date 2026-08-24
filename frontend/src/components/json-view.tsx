export function JsonView({ value }: { value: string }) {
  let pretty = value;
  try {
    pretty = JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    /* raw */
  }
  return (
    <pre className="overflow-auto rounded-md border border-border bg-[#0a0a0f] p-4 font-mono text-[13px] leading-6 text-zinc-300">
      {pretty || '(empty)'}
    </pre>
  );
}

export function jsonViewPretty(value: string): string {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}
