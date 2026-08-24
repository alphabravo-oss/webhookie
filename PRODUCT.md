# Webhookie

> **Alpha.** Experimental. **Use at your own risk.** Not for production.

Webhookie is the Mailpit of webhooks: a single binary (and Docker image) that pretends to be Slack, Microsoft Teams, PagerDuty, Discord, Telegram, Google Chat, Mattermost, Opsgenie, and generic HTTP so your app can send notifications without hitting the real internet — and that can fire signed fixtures at your receiver.

It is a **local and CI destination emulator**. It is not a production delivery gateway.

## Sinks vs destinations vs sources

- **Sinks** (inbound): your app POSTs to webhookie. We validate documented public-API rules (limits, enums, required children, Discord multipart files, Telegram HTML/Markdown/MarkdownV2, Adaptive Card element types; extra fields allowed), return the provider’s success/error envelope, and store the packet with JSON-pointer errors. Not Slack’s private validator and not Adaptive Card `additionalProperties: false`.
- **Destinations**: operator UIs for Slack / Teams / Discord / Mattermost / Telegram / Google Chat (channels, chats, spaces) and PagerDuty / Opsgenie (services, teams). Each channel has a stable webhook URL. Messages render in a generic version of the tool. Button / Ack / Resolve / Close clicks record an interaction and optionally POST a provider-shaped payload to an Interactivity URL you set. The transcript then updates **locally**; we do not apply the handler’s HTTP response to the message.
- **Inbox**: the packet log. Always the source of truth for what was captured.
- **Sources** (outbound): webhookie POSTs a signed fixture at your app so you can test handlers without Stripe/GitHub/Slack accounts.

## v1 providers

| Direction | Providers |
|---|---|
| Sinks | Generic HTTP bin, Slack incoming webhooks, Discord incoming webhooks, Teams MessageCard + Adaptive Card, PagerDuty Events API v2 (enqueue + change, ack/resolve), Telegram Bot `sendMessage`, Google Chat incoming webhooks, Mattermost incoming webhooks, Opsgenie Alert API v2 (create + acknowledge/close) |
| Sources | Standard Webhooks (`standard`), GitHub, Slack Events API (`slack-events`), Stripe, PagerDuty webhooks v3 (`pagerduty-v3`) |

## Default URLs

- Slack: `http://localhost:8080/hooks/slack/services/T00000000/B00000000/webhookie`
- Teams: `http://localhost:8080/hooks/teams/workflow/webhookie`
- PagerDuty: `http://localhost:8080/hooks/pagerduty/v2/enqueue` (routing key `0123456789abcdef0123456789abcdef`)
- Discord: `http://localhost:8080/hooks/discord/api/webhooks/0/webhookie`
- Telegram: `http://localhost:8080/hooks/telegram/bot/123456:AAWebhookie/sendMessage`
- Google Chat: `http://localhost:8080/hooks/googlechat/v1/spaces/AAAAwebhookie/messages`
- Mattermost: `http://localhost:8080/hooks/mattermost/hooks/mattermost-webhookie`
- Opsgenie: `http://localhost:8080/hooks/opsgenie/v2/alerts`
- Generic: `http://localhost:8080/hooks/generic/default`

New destination channels mint unique paths. See [docs/sinks.md](docs/sinks.md).

## Assertion API

- `GET /api/v1/events?provider=slack&contains=` (`contains` matches body text)
- `GET /api/v1/events/{id}`
- `DELETE /api/v1/events`
- `POST /api/v1/send`
- `GET /api/v1/events/stream` (SSE)

Go: `examples/go` `WaitFor` / `Reset`.

## Health

- `GET /healthz` liveness
- `GET /readyz` SQLite ping
- `GET /metrics` Prometheus text

## Auth

None by default. Optional `WEBHOOKIE_PASSWORD` protects the UI and `/api/v1/*` (Basic user `webhookie`, cookie session, or `access_token` query). **`/hooks/*` is never authenticated.**

## Non-goals

Production delivery, Zapier/workflows, Slack Bolt (slash commands, modals, `response_url` consume), Teams Bot Framework / Graph, Discord Interactions follow-up API, Telegram Bot API beyond `sendMessage`, public SaaS, Postgres, in-process tunnels.

## Run

Build the image yourself; no published registry image is assumed.

```bash
docker build -t webhookie:latest .
docker run --rm -p 8080:8080 -e WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080 -v webhookie-data:/data webhookie:latest
```

Docs: [docs/README.md](docs/README.md). License: Apache-2.0.
