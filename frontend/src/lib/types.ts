export type ValidationError = { path: string; message: string };

export type Chaos = {
  delayMs: number;
  status: number;
  body: string;
  contentType: string;
  hang: boolean;
};

export type Event = {
  id: string;
  sinkId: string;
  provider: string;
  receivedAt: string;
  method: string;
  path: string;
  query: Record<string, string>;
  headers: Record<string, string[]>;
  contentType: string;
  body: string;
  bodyTruncated: boolean;
  status: number;
  latencyMs: number;
  valid: boolean;
  validationErrors: ValidationError[];
  summary: string;
  groupKey: string;
};

export type Sink = {
  id: string;
  provider: string;
  name: string;
  token: string;
  path: string;
  url?: string;
  chaos: Chaos;
  createdAt: string;
};

export type Fixture = {
  provider: string;
  event: string;
  description: string;
};

export type SendAttempt = {
  id: string;
  createdAt: string;
  provider: string;
  eventName: string;
  target: string;
  status: number | null;
  error?: string;
  latencyMs: number;
};

export type Paginated<T> = {
  data: T;
  pagination?: { total: number; limit: number; offset: number; hasMore: boolean; nextOffset: number };
};

export type Envelope<T> = { data: T };

export type Channel = {
  id: string;
  workspaceId: string;
  sinkId: string;
  name: string;
  slug: string;
  kind: string;
  path: string;
  url: string;
  token: string;
  createdAt: string;
};

export type Workspace = {
  id: string;
  provider: string;
  name: string;
  interactivityUrl: string;
  signingSecret: string;
  createdAt: string;
  channels: Channel[];
};

export type Interaction = {
  id: string;
  createdAt: string;
  eventId: string;
  kind: string;
  actionId: string;
  payload: string;
  target: string;
  status: number | null;
  error?: string;
};

export type TakenAction = {
  actionId?: string;
  text?: string;
  kind?: string;
  pending?: boolean;
};
