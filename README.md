<p align="center">
  <img src="docs/logo.png" width="96" height="96" alt="Webhookie fishing-hook logo">
</p>

# Webhookie

> **Alpha 0.1.0.** Experimental software. APIs, URLs, and behavior will change. **Use at your own risk.** Not for production delivery.

The Mailpit of webhooks. One process that **emulates destination APIs** (Slack, Teams, Discord, PagerDuty, Telegram, Google Chat, Mattermost, Opsgenie, generic HTTP) so your app can POST notifications without hitting the real internet — and that can **fire signed fixtures** at your receiver.

It is a local/CI destination emulator, not a production gateway and not a clone of Slack.

```bash
docker run --rm -p 8080:8080 \
  -e WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080 \
  -v webhookie-data:/data \
  ghcr.io/alphabravo-oss/webhookie:0.1.0
```

Open http://localhost:8080

## What it is

| Face | What you use it for |
|---|---|
| **Sinks** | Your app POSTs to a provider-shaped URL. Webhookie validates documented public-API rules, returns the provider success/error envelope, and stores the packet plus JSON-pointer errors. |
| **Destinations** | Operator UIs (`/slack`, `/teams`, …) with channels/chats/spaces/services. Copy a URL, see the message, click Approve/Ack. |
| **Inbox** | Packet log for every capture. Schema errors, headers, original `body`, replay. Destination edits (`displayBody` / `deleted`) do not rewrite Inbox. |
| **Sources** | Webhookie POSTs a signed fixture at *your* app (`/api/v1/send`). |

Two-way: a destination click records an interaction and, if you set an Interactivity URL, POSTs a provider-shaped payload to your app. Click chrome (✓, hidden buttons) is local and does **not** apply your handler’s HTTP body. If the handler then calls the documented follow-up URL, webhookie **does** apply that to the original message (`displayBody` / `deleted`):

| Follow-up | Path |
|---|---|
| Slack `response_url` | `POST /hooks/slack/response/{eventId}` (`replace_original`, `delete_original`, or a new message) |
| Discord webhook message | `PATCH`/`DELETE /hooks/discord/api/webhooks/{id}/{token}/messages/{eventId}` — use `?wait=true` on execute so the id is returned |
| Telegram | `POST /hooks/telegram/bot/{token}/answerCallbackQuery` with `callback_query_id` = interaction id |

Details: [docs/](docs/README.md), [docs/sinks.md](docs/sinks.md), [docs/destinations.md](docs/destinations.md).

## Sink URLs (defaults)

Replace `http://localhost:8080` with `WEBHOOKIE_PUBLIC_BASE_URL` in Docker/CI.

| Provider | Default URL | Success |
|---|---|---|
| Generic | `/hooks/generic/default` | `{"ok":true}` `200` |
| Slack incoming | `/hooks/slack/services/T00000000/B00000000/webhookie` | `ok` `200` text/plain |
| Teams workflow | `/hooks/teams/workflow/webhookie` | MessageCard `1` or Adaptive `{statusCode:200}` |
| Discord incoming | `/hooks/discord/api/webhooks/0/webhookie` | `204`; `?wait=true` → `200` `{"id":"<eventId>","content":"..."}` |
| PagerDuty Events API v2 | `/hooks/pagerduty/v2/enqueue` | `202` + `dedup_key`. Routing key `0123456789abcdef0123456789abcdef` |
| Telegram `sendMessage` | `/hooks/telegram/bot/123456:AAWebhookie/sendMessage` | `{ok:true,result:{...}}` `200`. Also `…/answerCallbackQuery` |
| Google Chat incoming | `/hooks/googlechat/v1/spaces/AAAAwebhookie/messages` | Message resource `200` |
| Mattermost incoming | `/hooks/mattermost/hooks/mattermost-webhookie` | `ok` `200` text/plain |
| Opsgenie Alert API v2 | `/hooks/opsgenie/v2/alerts` | `202`. Optional `Authorization: GenieKey eb243592-faa2-4ba2-a551-webhookie01` |

Creating a channel in a destination UI mints a **new** URL. Copy it from that screen or `GET /api/v1/sinks`.

```bash
curl -s -X POST http://localhost:8080/hooks/slack/services/T00000000/B00000000/webhookie \
  -H 'content-type: application/json' \
  -d '{"text":"deploy failed"}'
```

Payload shapes, limits, and error envelopes: [docs/sinks.md](docs/sinks.md). Invalid payloads still land in Inbox (`valid: false`, `validationErrors`).

## Docker

Published image: **`ghcr.io/alphabravo-oss/webhookie`** (public). Tags: `0.1.0`, `0.1`, `latest`. linux/amd64 and linux/arm64.

```bash
docker pull ghcr.io/alphabravo-oss/webhookie:0.1.0
docker run --rm -p 8080:8080 \
  -e WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080 \
  -v webhookie-data:/data \
  ghcr.io/alphabravo-oss/webhookie:0.1.0
```

Compose (pulls the same tag; `--build` builds from this tree):

```bash
docker compose -f deploy/docker-compose.yml up
```

Local build: `make docker-build` then `make docker-run-local`. Multi-stage `Dockerfile` (Node 22 → Go 1.26 → Alpine `nonroot`). Full notes: [docs/getting-started.md](docs/getting-started.md).

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
| `WEBHOOKIE_VERSION` | `0.1.0` |

Retention is whichever limit hits first. SQLite file: `$WEBHOOKIE_DATA_DIR/webhookie.db`.

## Validation

Sinks check **documented public API** rules and reject with the provider envelope. Extra fields are allowed. Pointer errors are on the stored event (`GET /api/v1/events/{id}` → `validationErrors`); Slack’s HTTP body is still only `invalid_payload`.

| Provider | Documented checks |
|---|---|
| Slack | `text`/`blocks`/`attachments`, Block Kit types, section/actions/header/image/video/file/rich_text limits; `response_url` 5× / 30 min |
| Discord | content/embeds/components/poll, embed limits, `multipart/form-data` `files[n]` + `payload_json`; empty `50006`, fields `50035`; webhook message PATCH/DELETE |
| Teams | MessageCard envelope; Adaptive Card element types and version gates (`Table` needs 1.5, …) |
| Telegram | `chat_id`+`text`; HTML / Markdown / MarkdownV2; `entities[]` UTF-16 offsets; `answerCallbackQuery` |
| PagerDuty | 32-char `routing_key`, trigger payload, severity enum, summary ≤1024 |
| Opsgenie | `message` ≤130, `priority` P1–P5, alias/description/tags limits |
| Google Chat | `text` ≤4096, `cards` / `cardsV2` shape |
| Mattermost | `text` ≤16383 or `attachments` |

Not claimed: Slack’s private incoming-webhook 400s, Adaptive Card `additionalProperties: false` (that would reject real `msteams` payloads), a byte-clone of Telegram’s C++ parser, Slack Bolt, Teams Bot Framework, Discord Interactions follow-ups, Telegram Bot API beyond `sendMessage` and `answerCallbackQuery`.

See [docs/sinks.md](docs/sinks.md) and [docs/destinations.md](docs/destinations.md).

## License

Apache-2.0. See `LICENSE`.
