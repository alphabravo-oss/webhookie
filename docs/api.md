# HTTP API

JSON envelopes: success `{"data": ...}`, lists also `{"pagination": {total, limit, offset, hasMore, nextOffset}}`, errors `{"error": {"code": "...", "message": "..."}}`. CamelCase JSON.

OpenAPI skeleton (paths, little schema): [`../api/openapi.yaml`](../api/openapi.yaml). Handlers are chi, not generated servers.

`/hooks/*` is **not** this API. See [sinks.md](sinks.md).

When `WEBHOOKIE_PASSWORD` is set, `/api/v1/*` except `POST /api/v1/login` requires auth (see [configuration.md](configuration.md)). `/hooks/*`, `/healthz`, `/readyz`, `/metrics` stay open.

## Meta and probes

| Method | Path | Notes |
|---|---|---|
| GET | `/healthz` | `{"status":"ok"}` |
| GET | `/readyz` | SQLite ping; `503` if not |
| GET | `/metrics` | Prometheus text |
| GET | `/api/v1/meta` | `{version, publicBaseUrl}` |

## Events

| Method | Path | Notes |
|---|---|---|
| GET | `/api/v1/events` | Query: `provider`, `sinkId`, `since`, `groupKey`, `contains` (body LIKE), `limit`, `offset` |
| GET | `/api/v1/events/{id}` | `?unredact=1` to skip header redaction. Includes `valid` and `validationErrors` (`[{path, message}]`) |
| DELETE | `/api/v1/events` | All events |
| GET | `/api/v1/events/stream` | SSE. Event name `webhook`, data is the event JSON |
| POST | `/api/v1/events/{id}/replay` | Body `{"target":"https://..."}` |

Each event has `valid` (bool) and `validationErrors` (JSON pointers from the sink). The HTTP status on the original `/hooks/*` POST is `status`. Slack still returns only `invalid_payload` to the sender; the pointers are here and in Inbox.

Go test helpers (`examples/go`): `Reset(ctx, base)` → `DELETE /api/v1/events`; `WaitFor(ctx, base, provider, contains)` polls `GET /api/v1/events`.

## Sinks

| Method | Path | Notes |
|---|---|---|
| GET | `/api/v1/sinks` | Includes computed `url` |
| POST | `/api/v1/sinks` | Creates a **generic** bin (`{"name":"..."}`) |
| GET | `/api/v1/sinks/{id}` | |
| PATCH | `/api/v1/sinks/{id}` | `{"name":"...","chaos":{delayMs,status,body,contentType,hang}}` |

Chaos: `delayMs` sleeps first; `hang` waits until the client goes away; `status` 0 = no override; `status` 429 with empty body → body `rate_limited` and `Retry-After: 1`.

## Destinations (workspaces)

| Method | Path | Notes |
|---|---|---|
| GET | `/api/v1/workspaces` | All four original + Telegram, Google Chat, Mattermost, Opsgenie |
| GET | `/api/v1/workspaces/{id}` | Id (`ws-slack`) or provider (`slack`) |
| PATCH | `/api/v1/workspaces/{id}` | `{name, interactivityUrl, signingSecret}` |
| POST | `/api/v1/workspaces/{id}/channels` | `{name}` → new sink + channel |
| GET | `/api/v1/workspaces/{id}/channels/{channelId}/interactions` | Last 100 |
| POST | `/api/v1/workspaces/{id}/channels/{channelId}/actions` | See [destinations.md](destinations.md) |

## Sources

| Method | Path | Notes |
|---|---|---|
| GET | `/api/v1/fixtures` | `{provider, event, description}` |
| POST | `/api/v1/send` | `{provider, event, target, secret, timestampSkewSec}` |
| GET | `/api/v1/send/attempts` | |

Provider ids: `standard`, `github`, `slack-events`, `stripe`, `pagerduty-v3`.
