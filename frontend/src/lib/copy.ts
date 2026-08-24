export function eventToCurl(origin: string, ev: { method: string; path: string; headers: Record<string, string[]>; body: string }): string {
  const headers = Object.entries(ev.headers)
    .filter(([k]) => !['content-length', 'host', 'connection'].includes(k.toLowerCase()))
    .map(([k, vs]) => `-H '${k}: ${vs[0] ?? ''}'`)
    .join(' \\\n  ');
  const body = ev.body ? ` \\\n  --data-raw '${ev.body.replace(/'/g, `'\\''`)}'` : '';
  return `curl -s -X ${ev.method} ${origin}${ev.path} \\\n  ${headers}${body}`;
}
