export type WebhookEvent = {
  id: string;
  receivedAt: string;
  provider: 'generic' | 'slack' | 'discord' | 'teams' | 'pagerduty';
  method: string;
  path: string;
  status: number;
  latencyMs: number;
  valid: boolean;
  summary: string;
};

export type Sink = {
  id: string;
  provider: WebhookEvent['provider'];
  name: string;
  path: string;
  events: number;
};

export const demoSinks: Sink[] = [
  { id: 'sink-generic', provider: 'generic', name: 'Generic bin', path: '/hooks/generic/default', events: 12 },
  { id: 'sink-slack', provider: 'slack', name: 'Slack incoming', path: '/hooks/slack/services/T00000000/B00000000/webhookie', events: 4 },
  { id: 'sink-teams', provider: 'teams', name: 'Teams workflow', path: '/hooks/teams/workflow/webhookie', events: 2 },
  { id: 'sink-pagerduty', provider: 'pagerduty', name: 'PagerDuty Events API v2', path: '/hooks/pagerduty/v2/enqueue', events: 3 },
  { id: 'sink-discord', provider: 'discord', name: 'Discord webhook', path: '/hooks/discord/api/webhooks/0/webhookie', events: 1 },
];

export const demoEvents: WebhookEvent[] = [
  {
    id: 'evt-1',
    receivedAt: new Date(Date.now() - 12_000).toISOString(),
    provider: 'slack',
    method: 'POST',
    path: '/hooks/slack/services/T00000000/B00000000/webhookie',
    status: 200,
    latencyMs: 4,
    valid: true,
    summary: 'deploy failed on sierra-prod',
  },
  {
    id: 'evt-2',
    receivedAt: new Date(Date.now() - 48_000).toISOString(),
    provider: 'pagerduty',
    method: 'POST',
    path: '/hooks/pagerduty/v2/enqueue',
    status: 202,
    latencyMs: 6,
    valid: true,
    summary: 'trigger · disk usage > 90%',
  },
  {
    id: 'evt-3',
    receivedAt: new Date(Date.now() - 95_000).toISOString(),
    provider: 'generic',
    method: 'POST',
    path: '/hooks/generic/default',
    status: 200,
    latencyMs: 2,
    valid: true,
    summary: '{"ping":true}',
  },
  {
    id: 'evt-4',
    receivedAt: new Date(Date.now() - 140_000).toISOString(),
    provider: 'slack',
    method: 'POST',
    path: '/hooks/slack/services/T00000000/B00000000/webhookie',
    status: 400,
    latencyMs: 3,
    valid: false,
    summary: 'missing text and blocks',
  },
  {
    id: 'evt-5',
    receivedAt: new Date(Date.now() - 210_000).toISOString(),
    provider: 'teams',
    method: 'POST',
    path: '/hooks/teams/workflow/webhookie',
    status: 200,
    latencyMs: 8,
    valid: true,
    summary: 'Adaptive Card · scan complete',
  },
  {
    id: 'evt-6',
    receivedAt: new Date(Date.now() - 400_000).toISOString(),
    provider: 'discord',
    method: 'POST',
    path: '/hooks/discord/api/webhooks/0/webhookie',
    status: 204,
    latencyMs: 5,
    valid: true,
    summary: 'release v1.4.2 tagged',
  },
];

export const demoFixtures = [
  { provider: 'GitHub', event: 'push', description: 'Signed push event with X-Hub-Signature-256' },
  { provider: 'Slack Events', event: 'url_verification', description: 'Challenge handshake then event_callback' },
  { provider: 'Stripe', event: 'invoice.paid', description: 'Stripe-Signature t,v1 vector' },
  { provider: 'PagerDuty v3', event: 'incident.triggered', description: 'Outbound PD webhook v3' },
  { provider: 'Standard Webhooks', event: 'generic.ping', description: 'whsec_ HMAC with webhook-id headers' },
];
