import type { Chaos, Channel, Envelope, Event, Fixture, Interaction, Paginated, SendAttempt, Sink, Workspace } from './types';

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  });
  const text = await res.text();
  const json = text ? JSON.parse(text) : {};
  if (!res.ok) {
    throw new Error(json?.error?.message ?? res.statusText);
  }
  return json as T;
}

export function listEvents(params: Record<string, string | number | undefined> = {}) {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') q.set(k, String(v));
  }
  const s = q.toString();
  return req<Paginated<Event[]>>(`/api/v1/events${s ? `?${s}` : ''}`);
}

export function getEvent(id: string, unredact = false) {
  return req<Envelope<Event>>(`/api/v1/events/${id}${unredact ? '?unredact=1' : ''}`);
}

export function deleteEvents() {
  return req<Envelope<{ ok: boolean }>>('/api/v1/events', { method: 'DELETE' });
}

export function listSinks() {
  return req<Envelope<Sink[]>>('/api/v1/sinks');
}

export function patchSink(id: string, chaos: Partial<Chaos>) {
  return req<Envelope<Sink>>(`/api/v1/sinks/${id}`, { method: 'PATCH', body: JSON.stringify({ chaos }) });
}

export function getMeta() {
  return req<Envelope<{ version: string; publicBaseUrl: string }>>('/api/v1/meta');
}

export function listFixtures() {
  return req<Envelope<Fixture[]>>('/api/v1/fixtures');
}

export function sendEvent(body: { provider: string; event: string; target: string; secret: string; timestampSkewSec?: number }) {
  return req<Envelope<SendAttempt>>('/api/v1/send', { method: 'POST', body: JSON.stringify(body) });
}

export function listAttempts() {
  return req<Envelope<SendAttempt[]>>('/api/v1/send/attempts');
}

export function replayEvent(id: string, target: string) {
  return req<Envelope<SendAttempt>>(`/api/v1/events/${id}/replay`, { method: 'POST', body: JSON.stringify({ target }) });
}

export function getWorkspace(id: string) {
  return req<Envelope<Workspace>>(`/api/v1/workspaces/${id}`);
}

export function patchWorkspace(id: string, body: Partial<Pick<Workspace, 'interactivityUrl' | 'signingSecret' | 'name'>>) {
  return req<Envelope<Workspace>>(`/api/v1/workspaces/${id}`, { method: 'PATCH', body: JSON.stringify(body) });
}

export function createChannel(workspaceId: string, name: string) {
  return req<Envelope<Channel>>(`/api/v1/workspaces/${workspaceId}/channels`, { method: 'POST', body: JSON.stringify({ name }) });
}

export function listInteractions(workspaceId: string, channelId: string) {
  return req<Envelope<Interaction[]>>(`/api/v1/workspaces/${workspaceId}/channels/${channelId}/interactions`);
}

export function postAction(workspaceId: string, channelId: string, body: {
  eventId: string;
  kind: string;
  actionId?: string;
  value?: string;
  text?: string;
  blockId?: string;
}) {
  return req<Envelope<Interaction>>(`/api/v1/workspaces/${workspaceId}/channels/${channelId}/actions`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}
