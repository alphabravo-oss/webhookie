<p align="center">
  <img src="docs/logo.png" width="96" height="96" alt="Webhookie fishing-hook logo">
</p>

# Webhookie

> **Alpha.** Experimental software. APIs, URLs, and behavior will change. **Use at your own risk.** Not for production delivery.

The Mailpit of webhooks. One process that **emulates destination APIs** (Slack, Teams, Discord, PagerDuty, Telegram, Google Chat, Mattermost, Opsgenie, generic HTTP) so your app can POST notifications without hitting the real internet — and that can **fire signed fixtures** at your receiver.

It is a local/CI destination emulator, not a production gateway and not a clone of Slack.

```bash
make docker-run
# or: docker compose -f deploy/docker-compose.yml up --build
```

Open http://localhost:8080

## What it is

| Face | What you use it for |
|---|---|
| **Sinks** | Your app POSTs to a provider-shaped URL. Webhookie validates the payload (shallow), returns the real success/error envelope, and stores the packet. |
| **Destinations** | Operator UIs (`/slack`, `/teams`, …) with channels/chats/spaces/services. Copy a URL, see the message, click Approve/Ack. |
| **Inbox** | Packet log for every capture. Schema errors, headers, body, replay. |
| **Sources** | Webhookie POSTs a signed fixture at *your* app (`/api/v1/send`). |

Two-way clicks record an interaction and, if you set an Interactivity URL, POST a provider-shaped payload to your app. The destination UI then updates **locally**. Slack `response_url` is advertised in the payload but **not implemented** — a handler that tries to replace the original message will get 404.

Details: [docs/](docs/README.md).

## Sink URLs (defaults)

Replace `http://localhost:8080` with `WEBHOOKIE_PUBLIC_BASE_URL` in Docker/CI.

| Provider | Default URL | Success |
|---|---|---|
| Generic | `/hooks/generic/default` | `{"ok":true}` `200` |
| Slack incoming | `/hooks/slack/services/T00000000/B00000000/webhookie` | `ok` `200` text/plain |
| Teams workflow | `/hooks/teams/workflow/webhookie` | MessageCard `1` or Adaptive `{statusCode:200}` |
| Discord incoming | `/hooks/discord/api/webhooks/0/webhookie` | `204` (`200` if `?wait=true`) |
| PagerDuty Events API v2 | `/hooks/pagerduty/v2/enqueue` | `202` + `dedup_key`. Routing key `0123456789abcdef0123456789abcdef` |
| Telegram `sendMessage` | `/hooks/telegram/bot/123456:AAWebhookie/sendMessage` | `{ok:true,result:{...}}` `200` |
| Google Chat incoming | `/hooks/googlechat/v1/spaces/AAAAwebhookie/messages` | Message resource `200` |
| Mattermost incoming | `/hooks/mattermost/hooks/mattermost-webhookie` | `ok` `200` text/plain |
| Opsgenie Alert API v2 | `/hooks/opsgenie/v2/alerts` | `202`. Optional `Authorization: GenieKey eb243592-faa2-4ba2-a551-webhookie01` |

Creating a channel in a destination UI mints a **new** URL. Copy it from that screen or `GET /api/v1/sinks`.

```bash
curl -s -X POST http://localhost:8080/hooks/slack/services/T00000000/B00000000/webhookie \
  -H 'content-type: application/json' \
  -d '{"text":"deploy failed"}'
```

Payload shapes and required fields: [docs/sinks.md](docs/sinks.md).

## Docker

There is a multi-stage `Dockerfile` (Node 22 frontend → Go 1.26 binary → Alpine, user `nonroot`, port 8080, volume `/data`). Compose is `deploy/docker-compose.yml`.

```bash
docker build -t webhookie:latest .
docker run --rm -p 8080:8080 \
  -e WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080 \
  -v webhookie-data:/data \
  webhookie:latest
```

No published GHCR image is assumed. Build locally. Full notes: [docs/getting-started.md](docs/getting-started.md).

## Dev (two processes)

```bash
# terminal 1
go run ./cmd/webhookie

# terminal 2
cd frontend && npm run dev
```

Vite on `:5173` proxies `/api` and `/hooks` to `:8080`. Production/Docker serves the embedded UI from the Go binary on `:8080`.

```bash
make test          # go test + frontend vitest
make build         # frontend dist + go binary → bin/webhookie
```

## CI assertion API

```
GET    /api/v1/events?provider=slack&contains=deploy
GET    /api/v1/events/{id}
DELETE /api/v1/events
POST   /api/v1/send
GET    /api/v1/events/stream     # SSE
GET    /healthz  /readyz  /metrics
```

`contains` matches **event body** text, not the summary. Go helpers: `examples/go` (`WaitFor`, `Reset`). HTTP reference: [docs/api.md](docs/api.md). OpenAPI skeleton: `api/openapi.yaml`.

## Chaos

```bash
curl -X PATCH http://localhost:8080/api/v1/sinks/sink-slack \
  -H 'content-type: application/json' \
  -d '{"chaos":{"delayMs":500,"status":429,"hang":false}}'
```

`hang: true` holds the request until the client disconnects. `status: 0` (default) means no override.

## Configuration

| Env | Default |
|---|---|
| `WEBHOOKIE_ADDR` | `:8080` |
| `WEBHOOKIE_DATA_DIR` | `./data` (Docker: `/data`) |
| `WEBHOOKIE_PUBLIC_BASE_URL` | `http://localhost:8080` |
| `WEBHOOKIE_MAX_BODY_BYTES` | `1048576` (1 MiB) |
| `WEBHOOKIE_RETENTION_DAYS` | `7` |
| `WEBHOOKIE_MAX_EVENTS` | `10000` |
| `WEBHOOKIE_PASSWORD` | unset (no auth). UI + `/api/v1/*` only; **never** `/hooks/*` |
| `WEBHOOKIE_VERSION` | `0.1.0-dev` |

Retention is whichever limit hits first. SQLite file: `$WEBHOOKIE_DATA_DIR/webhookie.db`.

## Limits (accurate)

Webhookie is not Slack/Teams/Telegram. Incoming-webhook happy paths work. It does **not** implement Slack Bolt, Teams Bot Framework, Telegram Bot API beyond `sendMessage`, Discord Interactions follow-ups, or Slack `response_url` handling. Validation is shallow (required fields, not full Block Kit / Adaptive Card schema). See [docs/destinations.md](docs/destinations.md).

## License

Apache-2.0. See `LICENSE`.
